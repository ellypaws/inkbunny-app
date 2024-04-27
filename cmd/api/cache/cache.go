package cache

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/labstack/echo/v4"
	units "github.com/labstack/gommon/bytes"
	"github.com/redis/go-redis/v9"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Cache interface {
	Get(key string) (*Item, error)
	Set(key string, item *Item, duration time.Duration) error
}

type Item struct {
	Blob       []byte `json:"blob,omitempty"`
	MimeType   string `json:"mime_type,omitempty"`
	hitCount   int
	lastAccess time.Time
}

func (item *Item) MarshalBinary() ([]byte, error) {
	if item.MimeType == echo.MIMEApplicationJSON {
		blob := item.Blob
		item.Blob = nil
		b, err := json.Marshal(item)
		if err != nil {
			return b, err
		}
		return bytes.Join(
			[][]byte{
				b[:len(b)-1],
				[]byte(`,"blob":`),
				blob,
				[]byte("}"),
			}, nil), nil
	}
	return json.Marshal(item)
}

func (item *Item) UnmarshalBinary(b []byte) error {
	return json.Unmarshal(b, item)
}

func (item *Item) Accessed() {
	backoff := int64(min(math.Pow(2, float64(item.hitCount-1)), 24*time.Hour.Seconds()))
	item.lastAccess = time.Now().Add(time.Duration(backoff) * time.Second)
	item.hitCount += 1
}

type q struct {
	ongoing map[string]chan *Item
	mu      sync.Mutex
}

var queue = q{ongoing: make(map[string]chan *Item)}

type Fetch struct {
	Key      string
	URL      string
	MimeType string
}

const (
	Indefinite = 0
	Now        = time.Nanosecond
	Day        = 24 * time.Hour
	Week       = 7 * Day
	Month      = 30 * Day
	Year       = 365 * Day
)

func Retrieve(c echo.Context, cache Cache, fetch Fetch) (*Item, func(c echo.Context) error) {
	parse, err := url.Parse(fetch.URL)
	if err != nil {
		c.Logger().Errorf("Failed to parse url: %s", err)
		return nil, ErrFunc(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if fetch.MimeType == "" {
		c.Logger().Warnf("no mime type provided for %s", fetch.URL)
		fetch.MimeType = MimeTypeFromURL(fetch.URL)
	}

	if strings.Contains(fetch.URL, "private_") {
		if sid, ok := c.Get("sid").(string); ok && sid != "" {
			q := parse.Query()
			if s := c.QueryParam("sid"); s != "" && s != sid {
				sid = s
				c.Logger().Warnf("using query parameter override for sid: %s", sid)
			}
			q.Set("sid", sid)
			parse.RawQuery = q.Encode()
			fetch.Key = fmt.Sprintf("%s:%s", fetch.MimeType, parse)
			fetch.URL = parse.String()
		} else {
			return nil, ErrFunc(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "Private files require a session ID"})
		}
	}

	if !strings.HasPrefix(fetch.Key, fetch.MimeType) {
		c.Logger().Warnf("key %s does not start with %s", fetch.Key, fetch.MimeType)
		fetch.Key = fmt.Sprintf("%s:%s", fetch.MimeType, fetch.URL)
	}

	if c.Request().Header.Get(echo.HeaderCacheControl) != "no-cache" {
		item, err := cache.Get(fetch.Key)
		if err == nil {
			c.Logger().Infof("Retrieved %s %dKiB", fetch.Key, len(item.Blob)/units.KiB)
			c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=86400")
			return item, nil
		}

		if !errors.Is(err, redis.Nil) && fetch.URL == "" {
			c.Logger().Errorf("could not get %s from cache %T", fetch.Key, cache)
			return nil, ErrFunc(http.StatusInternalServerError, err)
		}

		c.Logger().Debugf("Cache miss for %s retrieving image...", fetch.Key)
	}

	queue.mu.Lock()
	if ongoing, found := queue.ongoing[fetch.Key]; found {
		queue.mu.Unlock()
		c.Logger().Warnf("Still receiving %s", fetch.URL)
		item := <-ongoing
		item, err := cache.Get(fetch.Key)
		if err != nil {
			c.Logger().Errorf("could not get %s from cache %T: %v", fetch.Key, cache, err)
			return nil, ErrFunc(http.StatusInternalServerError, err)
		}
		c.Logger().Infof("Retrieved %s %s %dKiB", fetch.Key, item.MimeType, len(item.Blob)/units.KiB)
		c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=86400") // 24 hours
		return item, nil
	}

	done := make(chan *Item)
	queue.ongoing[fetch.Key] = done
	queue.mu.Unlock()
	defer func() {
		queue.mu.Lock()
		close(done)
		delete(queue.ongoing, fetch.Key)
		queue.mu.Unlock()
	}()

	c.Logger().Infof("Downloading %s", fetch.URL)
	resp, err := http.Get(fetch.URL)
	if err != nil {
		c.Logger().Errorf("failed to fetch resource %v, %v", fetch.URL, err)
		return nil, ErrFunc(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: fmt.Sprintf("failed to fetch resource %v", fetch.URL), Debug: err})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Logger().Errorf("unexpected status code %d", resp.StatusCode)
		return nil, ErrFunc(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: fmt.Sprintf("unexpected status code %d", resp.StatusCode)})
	}

	blob, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Logger().Errorf("could not read body %v", err)
		return nil, ErrFunc(http.StatusInternalServerError, crashy.Wrap(err))
	}

	mimeType := resp.Header.Get(echo.HeaderContentType)

	if mimeType == "" {
		mimeType = MimeTypeFromURL(fetch.URL)
	}

	if mimeType != fetch.MimeType {
		c.Logger().Warnf(`mismatched mime types expected: "%s" got: "%s"`, fetch.MimeType, mimeType)
	}

	if bytes.HasPrefix(blob, []byte("ERROR")) {
		c.Logger().Errorf("error downloading %s: %s", fetch.URL, blob)
		return nil, ErrFunc(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: fmt.Sprintf("error downloading %s: %s", fetch.URL, blob)})
	}

	item := &Item{
		Blob:     blob,
		MimeType: mimeType,
	}

	err = cache.Set(fmt.Sprintf("%v:%v", item.MimeType, fetch.URL), item, Day)
	if err != nil {
		c.Logger().Errorf("could not set %s in cache %T: %v", fetch.URL, cache, err)
	}
	c.Logger().Infof("Cached %s %dKiB", fetch.Key, len(item.Blob)/units.KiB)

	c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=86400") // 24 hours

	return item, nil
}

func MimeTypeFromURL(url string) string {
	split := strings.Split(url, ".")
	mimeType := mime.TypeByExtension("." + split[len(split)-1])
	return mimeType
}

func KeyWithMimeType(url string) string {
	return fmt.Sprintf("%s:%s", MimeTypeFromURL(url), url)
}

func ErrFunc(r int, err error) func(c echo.Context) error {
	return func(c echo.Context) error {
		return c.JSON(r, crashy.Wrap(err))
	}
}

func SwitchCache(c echo.Context) Cache {
	var cacheToUse Cache
	r, ok := c.Get("redis").(*Redis)
	if ok {
		cacheToUse = r
	} else {
		cacheToUse = GetLocalCache(c)
	}
	return cacheToUse
}

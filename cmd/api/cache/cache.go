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
	"strings"
	"sync"
	"time"
)

type Cache interface {
	Get(key string) (*Item, error)
	Set(key string, item *Item) error
}

type Item struct {
	Blob       []byte    `json:"blob,omitempty"`
	LastAccess time.Time `json:"last_access"`
	MimeType   string    `json:"mime_type,omitempty"`
	HitCount   int       `json:"hit_count,omitempty"`
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
	backoff := int64(min(math.Pow(2, float64(item.HitCount-1)), 24*time.Hour.Seconds()))
	item.LastAccess = time.Now().Add(time.Duration(backoff) * time.Second)
	item.HitCount += 1
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

func Retrieve(c echo.Context, cache Cache, fetch Fetch) (*Item, func(c echo.Context) error) {
	if !strings.HasPrefix(fetch.Key, fetch.MimeType) {
		fetch.Key = fmt.Sprintf("%v:%v", fetch.MimeType, fetch.URL)
	}

	item, err := cache.Get(fetch.Key)
	if err == nil {
		c.Logger().Infof("Retrieved %s %s %dKiB", fetch.URL, item.MimeType, len(item.Blob)/units.KiB)
		c.Response().Header().Set("Cache-Control", "public, max-age=86400")
		return item, nil
	}

	if !errors.Is(err, redis.Nil) && fetch.URL == "" {
		c.Logger().Errorf("could not get %s from cache %T", fetch.Key, cache)
		return nil, errFunc(http.StatusInternalServerError, err)
	}

	c.Logger().Debugf("Cache miss for %s retrieving image...", fetch.Key)

	queue.mu.Lock()
	if ongoing, found := queue.ongoing[fetch.Key]; found {
		queue.mu.Unlock()
		c.Logger().Debugf("Still receiving %s", fetch.URL)
		item := <-ongoing
		item, err := cache.Get(fetch.Key)
		if err != nil {
			c.Logger().Errorf("could not get %s from cache %T: %v", fetch.Key, cache, err)
			return nil, errFunc(http.StatusInternalServerError, err)
		}
		c.Logger().Infof("Retrieved %s %s %dKiB", fetch.URL, item.MimeType, len(item.Blob)/units.KiB)
		c.Response().Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
		return item, nil
	}

	done := make(chan *Item)
	queue.ongoing[fetch.Key] = done
	queue.mu.Unlock()

	c.Logger().Infof("Downloading %s", fetch.URL)
	resp, err := http.Get(fetch.URL)
	if err != nil {
		c.Logger().Errorf("failed to fetch resource %v, %v", fetch.URL, err)
		return nil, errFunc(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: fmt.Sprintf("failed to fetch resource %v", fetch.URL), Debug: err})
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Logger().Errorf("unexpected status code %d", resp.StatusCode)
		return nil, errFunc(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: fmt.Sprintf("unexpected status code %d", resp.StatusCode)})
	}

	blob, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Logger().Errorf("could not read body %v", err)
		return nil, errFunc(http.StatusInternalServerError, crashy.Wrap(err))
	}

	mimeType := resp.Header.Get("Content-Type")

	if mimeType == "" {
		mimeType = MimeTypeFromURL(fetch.URL)
	}

	item = &Item{
		Blob:       blob,
		LastAccess: time.Now().UTC(),
		MimeType:   mimeType,
	}

	err = cache.Set(fmt.Sprintf("%v:%v", mimeType, fetch.URL), item)
	if err != nil {
		c.Logger().Errorf("could not set %s in cache %T: %v", fetch.URL, cache, err)
	}
	c.Logger().Infof("Cached %s %s %dKiB", fetch.Key, item.MimeType, len(item.Blob)/units.KiB)

	queue.mu.Lock()
	close(done)
	delete(queue.ongoing, fetch.Key)
	queue.mu.Unlock()

	c.Response().Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
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

func errFunc(r int, err error) func(c echo.Context) error {
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

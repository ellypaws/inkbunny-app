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

func Retrieve(c echo.Context, cache Cache, key string, url string) (*Item, func(c echo.Context) error) {
	item, err := cache.Get(key)
	if err == nil {
		c.Logger().Infof("Retrieved %s %s %dKiB", url, item.MimeType, len(item.Blob)/units.KiB)
		c.Response().Header().Set("Cache-Control", "public, max-age=86400")
		return item, nil
	}

	if !errors.Is(err, redis.Nil) && url == "" {
		c.Logger().Errorf("could not get %s from cache %T", key, cache)
		return nil, errFunc(http.StatusInternalServerError, err)
	}

	queue.mu.Lock()
	if ongoing, found := queue.ongoing[key]; found {
		queue.mu.Unlock()
		c.Logger().Debugf("Still receiving %s", url)
		item := <-ongoing
		item, err := cache.Get(key)
		if err != nil {
			c.Logger().Errorf("could not get %s from cache %T: %v", key, cache, err)
			return nil, errFunc(http.StatusInternalServerError, err)
		}
		c.Logger().Infof("Retrieved %s %s %dKiB", url, item.MimeType, len(item.Blob)/units.KiB)
		c.Response().Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
		return item, nil
	}

	done := make(chan *Item)
	queue.ongoing[key] = done
	queue.mu.Unlock()

	c.Logger().Infof("Downloading %s", url)
	resp, err := http.Get(url)
	if err != nil {
		c.Logger().Errorf("failed to fetch resource %v, %v", url, err)
		return nil, errFunc(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: fmt.Sprintf("failed to fetch resource %v", url), Debug: err})
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
		mimeType = MimeType(url)
	}

	item = &Item{
		Blob:       blob,
		LastAccess: time.Now().UTC(),
		MimeType:   mimeType,
	}

	err = cache.Set(fmt.Sprintf("%v:%v", mimeType, url), item)
	if err != nil {
		c.Logger().Errorf("could not set %s in cache %T: %v", url, cache, err)
	}
	c.Logger().Infof("Cached %s %s %dKiB", key, item.MimeType, len(item.Blob)/units.KiB)

	queue.mu.Lock()
	close(done)
	delete(queue.ongoing, key)
	queue.mu.Unlock()

	c.Response().Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
	return item, nil
}

func MimeType(url string) string {
	split := strings.Split(url, ".")
	mimeType := mime.TypeByExtension("." + split[len(split)-1])
	return mimeType
}

func MimeTypeURL(url string) string {
	return fmt.Sprintf("%s:%s", MimeType(url), url)
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

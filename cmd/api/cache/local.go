package cache

import (
	"errors"
	"fmt"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/bytes"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var FileCache = &LocalCache{
	items:    make(map[string]*Item),
	ongoing:  make(map[string]chan *Item),
	maxSize:  256 * bytes.MiB,
	maxItems: 20,
}

var TextCache = &LocalCache{
	items:    make(map[string]*Item),
	ongoing:  make(map[string]chan *Item),
	maxSize:  32 * bytes.MiB,
	maxItems: 256,
}

type LocalCache struct {
	items       map[string]*Item
	ongoing     map[string]chan *Item
	maxSize     int64 // Max size in bytes
	maxItems    int
	currentSize int64
	mu          sync.Mutex
}

var UrlNotString = errors.New("url is set but cannot be coerced into string")
var UrlNotSet = errors.New("url is not set")
var StatusNotOK = errors.New("unexpected status code")

// Get downloads and sets the cache
func (l *LocalCache) Get(c echo.Context, url string) (*Item, error) {
	cacheItem, err := l.get(url)
	if err == nil {
		c.Logger().Infof("Cache hit for %s", url)
		c.Response().Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
		return cacheItem, nil
	}

	l.mu.Lock()
	if ongoing, found := l.ongoing[url]; found {
		l.mu.Unlock()
		c.Logger().Debugf("Still receiving %s", url)
		item := <-ongoing
		item, err := l.get(url)
		if err != nil {
			return nil, err
		}
		c.Logger().Debugf("Retrieved %s %s %dKiB", url, item.MimeType, len(item.Blob)/bytes.KiB)
		c.Response().Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
		return item, nil
	}

	c.Logger().Infof("Downloading %s", url)

	done := make(chan *Item)
	l.ongoing[url] = done
	l.mu.Unlock()

	resp, err := http.Get(url)
	if err != nil {
		return nil, crashy.ErrorResponse{ErrorString: fmt.Sprintf("failed to fetch resource %v", url), Debug: err}
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", StatusNotOK, resp.StatusCode)
	}

	blob, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, crashy.ErrorResponse{ErrorString: "failed to read data", Debug: err}
	}

	mimeType := resp.Header.Get("Content-Type")
	item := &Item{
		Blob:       blob,
		LastAccess: time.Now().UTC(),
		MimeType:   mimeType,
	}
	err = l.Set(c, url, item)
	if err != nil {
		c.Logger().Errorf("could not set %s in cache %T", url, l)
	}
	c.Logger().Debugf("Cached %s %s %dKiB", url, mimeType, len(blob)/bytes.KiB)
	c.Response().Header().Set("Cache-Control", "public, max-age=86400")

	return item, nil
}

func (l *LocalCache) Set(c echo.Context, key string, item *Item) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, found := l.items[key]; !found {
		if len(l.items) >= l.maxItems || (l.currentSize) > l.maxSize {
			l.Evict()
		}
		l.items[key] = item
		l.currentSize += int64(len(item.Blob))
	}
	if channel, found := l.ongoing[key]; found && channel != nil {
		close(channel)
		delete(l.ongoing, key)
	}

	return nil
}

func GetLocalCache(c echo.Context) *LocalCache {
	accept := c.Request().Header.Get("Accept")

	if strings.HasPrefix(accept, "text") {
		return TextCache
	}

	if strings.HasSuffix(accept, "json") {
		return TextCache
	}

	return FileCache
}

var ErrNoItem = errors.New("no such key")

func (l *LocalCache) get(key string) (*Item, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	item, found := l.items[key]
	if found {
		item.Accessed()
		return item, nil
	}
	return nil, ErrNoItem
}

func (l *LocalCache) Evict() {
	now := time.Now()
	for k, v := range l.items {
		// Check if an item is past its expiration time
		if now.After(v.LastAccess) {
			delete(l.items, k)
			l.currentSize -= int64(len(v.Blob))
		}
	}

	// If still over capacity, remove least recently accessed items
	for (len(l.items) > l.maxItems || l.currentSize > l.maxSize) && len(l.items) > 0 {
		var oldestKey string
		var oldestItem *Item
		for k, v := range l.items {
			if oldestItem == nil || v.LastAccess.Before(oldestItem.LastAccess) {
				oldestKey = k
				oldestItem = v
			}
		}
		if oldestItem != nil { // Found an item to remove
			delete(l.items, oldestKey)
			l.currentSize -= int64(len(oldestItem.Blob))
		}
	}
}

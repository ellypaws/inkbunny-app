package cache

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/bytes"
	"github.com/redis/go-redis/v9"
	"strings"
	"sync"
	"time"
)

var FileCache = &LocalCache{
	items:    make(map[string]*Item),
	maxSize:  256 * bytes.MiB,
	maxItems: 20,
}

var TextCache = &LocalCache{
	items:    make(map[string]*Item),
	maxSize:  32 * bytes.MiB,
	maxItems: 256,
}

type LocalCache struct {
	items       map[string]*Item
	maxSize     int64 // Max size in bytes
	maxItems    int
	currentSize int64
	mu          sync.Mutex
}

var UrlNotString = errors.New("url is set but cannot be coerced into string")
var UrlNotSet = errors.New("url is not set")
var StatusNotOK = errors.New("unexpected status code")

func (l *LocalCache) Get(key string) (*Item, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	item, found := l.items[key]
	if found {
		item.Accessed()
		return item, nil
	}
	return nil, redis.Nil
}

func (l *LocalCache) Set(key string, item *Item, duration time.Duration) error {
	if !strings.HasPrefix(key, item.MimeType) {
		key = fmt.Sprintf("%s:%s", item.MimeType, key)
	}

	item.LastAccess = time.Now().UTC().Add(-duration)

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, found := l.items[key]; !found {
		if len(l.items) >= l.maxItems || (l.currentSize) > l.maxSize {
			l.Evict()
		}
		l.items[key] = item
		l.currentSize += int64(len(item.Blob))
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

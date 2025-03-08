package cache

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/bytes"
	"github.com/redis/go-redis/v9"
)

var FileCache = &LocalCache{
	items:    make(map[string]*Item),
	maxSize:  256 * bytes.MiB,
	maxItems: 1024,
}

var TextCache = &LocalCache{
	items:    make(map[string]*Item),
	maxSize:  256 * bytes.MiB,
	maxItems: 2048,
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

func (l *LocalCache) MGet(keys ...string) (map[string]*Item, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	items := make(map[string]*Item)
	for _, key := range keys {
		if item, found := l.items[key]; found {
			item.Accessed()
			items[key] = item
		} else {
			items[key] = nil
		}
	}
	return items, nil
}

func (l *LocalCache) Set(key string, item *Item, duration time.Duration) error {
	if !strings.HasPrefix(key, item.MimeType) {
		key = fmt.Sprintf("%s:%s", item.MimeType, key)
	}

	item.lastAccess = time.Now().UTC().Add(-duration)

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
		if now.After(v.lastAccess) {
			delete(l.items, k)
			l.currentSize -= int64(len(v.Blob))
		}
	}

	// If still over capacity, remove least recently accessed items
	for (len(l.items) > l.maxItems || l.currentSize > l.maxSize) && len(l.items) > 0 {
		var oldestKey string
		var oldestItem *Item
		for k, v := range l.items {
			if oldestItem == nil || v.lastAccess.Before(oldestItem.lastAccess) {
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

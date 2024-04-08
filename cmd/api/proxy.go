package main

import (
	"fmt"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/bytes"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var fileCache = &FileCache{
	items:    make(map[string]*CacheItem),
	maxSize:  256 * bytes.MiB,
	maxItems: 20,
}

var textCache = &FileCache{
	items:    make(map[string]*CacheItem),
	maxSize:  32 * bytes.MiB,
	maxItems: 256,
}

type FileCache struct {
	items       map[string]*CacheItem
	maxSize     int64 // Max size in bytes
	maxItems    int
	currentSize int64
	mu          sync.Mutex
}

type CacheItem struct {
	Blob       []byte    // The image data
	LastAccess time.Time // Last access time
	MimeType   string    // MIME type of the image
	HitCount   int       // Number of accesses
}

// GetImageHandler handles image apis, caching them as needed.
func GetImageHandler(c echo.Context) error {
	imageURL := c.QueryParam("url")
	if imageURL == "" {
		return c.String(http.StatusBadRequest, "URL query parameter is required")
	}

	parse, err := url.Parse(imageURL)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Failed to parse url: %s", err))
	}

	hostname := parse.Hostname()
	valid := strings.HasSuffix(hostname, ".ib.metapix.net") || hostname == "inkbunny.net"
	if !valid {
		return c.String(http.StatusBadRequest, "URL must be from inkbunny.net")
	}

	key := parse.Path

	cacheItem, errorFunc := GetCache(c, key, imageURL)
	if errorFunc != nil {
		return errorFunc(c)
	}

	return c.Blob(http.StatusOK, cacheItem.MimeType, cacheItem.Blob)
}

func GetCache(c echo.Context, key string, fileURL string) (*CacheItem, func(c echo.Context) error) {
	cacheToUse := fileCache
	accept := c.Request().Header.Get("Accept")
	if strings.HasPrefix(accept, "text") {
		cacheToUse = textCache
	}
	if strings.HasSuffix(accept, "json") {
		cacheToUse = textCache
	}
	cacheItem, found := cacheToUse.Get(key)
	if found {
		c.Logger().Infof("Cache hit for %s", key)
		c.Response().Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
		return cacheItem, nil
	}

	c.Logger().Infof("Cache miss for %s, retrieving %s", key, fileURL)
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, func(c echo.Context) error {
			return c.JSON(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: "failed to fetch image", Debug: err})
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, func(c echo.Context) error {
			return c.NoContent(resp.StatusCode)
		}
	}

	blob, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, func(c echo.Context) error {
			return c.JSON(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: "failed to read data", Debug: err})
		}
	}

	mimeType := resp.Header.Get("Content-Type")
	item := &CacheItem{
		Blob:       blob,
		LastAccess: time.Now().Add(1 * time.Second),
		MimeType:   mimeType,
		HitCount:   1,
	}
	cacheToUse.Set(key, item)
	c.Logger().Debugf("Cached %s %s %dKiB", key, mimeType, len(blob)/bytes.KiB)
	c.Response().Header().Set("Cache-Control", "public, max-age=86400")
	return item, nil
}

func (c *FileCache) Get(key string) (*CacheItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, found := c.items[key]
	if found {
		backoff := int64(min(math.Pow(2, float64(item.HitCount-1)), 24*time.Hour.Seconds())) // max backoff of 24 hours
		item.LastAccess = time.Now().Add(time.Duration(backoff) * time.Second)
		item.HitCount += 1
		return item, true
	}
	return nil, false
}

func (c *FileCache) Set(key string, item *CacheItem) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, found := c.items[key]; !found {
		if len(c.items) >= c.maxItems || (c.currentSize) > c.maxSize {
			c.evict()
		}
		c.items[key] = item
		c.currentSize += int64(len(item.Blob))
	}
}

func (c *FileCache) evict() {
	now := time.Now()
	for k, v := range c.items {
		// Check if an item is past its expiration time
		if now.After(v.LastAccess) {
			delete(c.items, k)
			c.currentSize -= int64(len(v.Blob))
		}
	}

	// If still over capacity, remove least recently accessed items
	for (len(c.items) > c.maxItems || c.currentSize > c.maxSize) && len(c.items) > 0 {
		var oldestKey string
		var oldestItem *CacheItem
		for k, v := range c.items {
			if oldestItem == nil || v.LastAccess.Before(oldestItem.LastAccess) {
				oldestKey = k
				oldestItem = v
			}
		}
		if oldestItem != nil { // Found an item to remove
			delete(c.items, oldestKey)
			c.currentSize -= int64(len(oldestItem.Blob))
		}
	}
}

func handlePath(c echo.Context) error {
	path := c.Param("path")
	return ProxyHandler(path)(c)
}

func ProxyHandler(path string) echo.HandlerFunc {
	return func(c echo.Context) error {
		req, err := http.NewRequest(http.MethodPost, host.WithPath(path).String(), c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// Forward any headers if necessary
		for name, values := range c.Request().Header {
			for _, value := range values {
				req.Header.Add(name, value)
			}
		}

		// Execute the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadGateway, err.Error())
		}
		defer resp.Body.Close()

		// Copy the response headers and status code to the Echo context response
		for name, values := range resp.Header {
			for _, value := range values {
				c.Response().Header().Add(name, value)
			}
		}
		c.Response().WriteHeader(resp.StatusCode)

		// Stream the response body directly to the client without modification
		_, err = io.Copy(c.Response().Writer, resp.Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return nil
	}
}

package main

import (
	"fmt"
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

var imageCache = &ImageCache{
	items:       make(map[string]*CacheItem),
	maxSize:     10 * bytes.MB,
	maxItems:    20,
	currentSize: 0,
	mu:          sync.Mutex{},
}

// GetImageHandler handles image apis, caching them as needed.
func GetImageHandler(c echo.Context) error {
	imageURL := c.QueryParam("url")
	if imageURL == "" {
		return c.String(http.StatusBadRequest, "URL query parameter is required")
	}

	// parse url to url.Url
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

	cacheItem, found := imageCache.Get(key)
	if found {
		// Set Cache-Control header for browser caching
		c.Response().Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
		// Serve the blob from cache
		return c.Blob(http.StatusOK, "image/jpeg", cacheItem.Blob) // Adjust MIME type as necessary
	}

	// Image not in cache, fetch it
	resp, err := http.Get(imageURL)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch image: %s", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.String(http.StatusInternalServerError, "Failed to fetch image: invalid status code")
	}

	imgBlob, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to read image data: %s", err))
	}

	// Store in cache
	imageCache.Set(key, imgBlob)

	// Serve the fetched image
	return c.Blob(http.StatusOK, "image/jpeg", imgBlob) // Adjust MIME type as necessary
}

type CacheItem struct {
	Blob       []byte    // The image data
	LastAccess time.Time // Last access time
	HitCount   int       // Number of accesses
}

type ImageCache struct {
	items       map[string]*CacheItem
	maxSize     int64 // Max size in bytes
	maxItems    int
	currentSize int64
	mu          sync.Mutex
}

func (c *ImageCache) Get(key string) (*CacheItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, found := c.items[key]
	if found {
		backoff := int64(math.Min(math.Pow(2, float64(item.HitCount-1)), 300)) // exponential backoff up to 300 seconds
		item.LastAccess = time.Now().Add(time.Duration(backoff) * time.Second)
		item.HitCount += 1
		return item, true
	}
	return nil, false
}

func (c *ImageCache) Set(key string, blob []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, found := c.items[key]; !found {
		if len(c.items) >= c.maxItems || (c.currentSize) > c.maxSize {
			c.evict()
		}
		c.items[key] = &CacheItem{
			Blob:       blob,
			LastAccess: time.Now().Add(1 * time.Second), // initial backoff of 1 second
			HitCount:   1,
		}
		c.currentSize += int64(len(blob))
	}
}

func (c *ImageCache) evict() {
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

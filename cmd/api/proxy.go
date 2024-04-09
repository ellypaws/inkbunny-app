package main

import (
	"fmt"
	"github.com/ellypaws/inkbunny-app/api/cache"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"net/url"
	"strings"
)

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

	cacheItem, errorFunc := cache.Retrieve(c, cache.GetLocalCache(c), key)
	if errorFunc != nil {
		return errorFunc(c)
	}

	return c.Blob(http.StatusOK, cacheItem.MimeType, cacheItem.Blob)
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

func HandlePath(c echo.Context) error {
	path := c.Param("path")
	return ProxyHandler(path)(c)
}

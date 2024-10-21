package api

import (
	"fmt"
	"github.com/ellypaws/inkbunny-app/cmd/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/api/service"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
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
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{
			ErrorString: "URL must be from inkbunny.net or *.ib.metapix.net",
			Debug:       hostname,
		})
	}

	if strings.Contains(imageURL, "private_") {
		if sid, ok := c.Get("sid").(string); ok && sid != "" {
			q := parse.Query()
			if querySID := c.QueryParam("sid"); querySID != "" {
				sid = querySID
			}
			q.Set("sid", sid)
			parse.RawQuery = q.Encode()
		} else {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "Private files require a session ID"})
		}
	}

	mimeType := cache.MimeTypeFromURL(imageURL)
	cacheItem, errorFunc := cache.Retrieve(c, cache.SwitchCache(c), cache.Fetch{
		Key:      fmt.Sprintf("%s:%s", mimeType, parse),
		URL:      parse.String(),
		MimeType: mimeType,
	})
	if errorFunc != nil {
		return errorFunc(c)
	}

	return c.Blob(http.StatusOK, cacheItem.MimeType, cacheItem.Blob)
}

func ProxyHandler(path string) echo.HandlerFunc {
	return func(c echo.Context) error {
		req, err := http.NewRequest(http.MethodPost, SDHost.WithPath(path).String(), c.Request().Body)
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

// GetAvatarHandler returns the avatar URL of a user
func GetAvatarHandler(c echo.Context) error {
	username := c.Param("username")
	if username == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing username"})
	}

	cacheToUse := cache.SwitchCache(c)

	users, err := service.RetrieveUsers(c, username, true)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	item, errFunc := service.RetrieveAvatar(c, cacheToUse, users[0])
	if errFunc != nil {
		return errFunc(c)
	}
	return c.Blob(http.StatusOK, item.MimeType, item.Blob)
}

func HandlePath(c echo.Context) error {
	path := c.Param("path")
	return ProxyHandler(path)(c)
}

var divideThousand = regexp.MustCompile(`^(\d+?)\d{3}_`)

func GetFileHandler(c echo.Context) error {
	file := c.Param("file")
	if strings.Contains(file, "..") {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "invalid file path"})
	}
	fileName := filepath.Base(file)
	index := "0"
	divide := divideThousand.FindStringSubmatch(fileName)
	if len(divide) > 1 {
		index = divide[1]
	}
	return c.File(filepath.Join(".", "files", "full", index, fileName))
}

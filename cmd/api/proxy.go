package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ellypaws/inkbunny-app/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
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
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{
			ErrorString: "URL must be from inkbunny.net or *.ib.metapix.net",
			Debug:       hostname,
		})
	}

	if strings.Contains(imageURL, "private_files") {
		if sid, ok := c.Get("sid").(string); ok && sid != "" {
			q := parse.Query()
			c.Logger().Debugf("Setting sid: %s for private file %s", sid, parse)
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

// GetAvatarHandler returns the avatar URL of a user
func GetAvatarHandler(c echo.Context) error {
	username := c.Param("username")
	if username == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing username"})
	}

	cacheToUse := cache.SwitchCache(c)
	key := fmt.Sprintf("%v:inkbunny:username_autosuggest:exact:%v", echo.MIMEApplicationJSON, username)

	item, err := cacheToUse.Get(key)
	if err == nil {
		var users []api.Autocomplete
		if err := json.Unmarshal(item.Blob, &users); err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
		if len(users) == 0 {
			return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no users found"})
		}

		item, errFunc := cache.Retrieve(c, cacheToUse, cache.Fetch{
			URL:      fmt.Sprintf("https://jp.ib.metapix.net/usericons/small/%v", users[0].Icon),
			MimeType: cache.MimeTypeFromURL(users[0].Icon),
		})
		if errFunc != nil {
			return errFunc(c)
		}
		return c.Blob(http.StatusOK, item.MimeType, item.Blob)
	}
	if !errors.Is(err, redis.Nil) {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "an error occurred while retrieving the username", Debug: err})
	}

	usernames, err := api.GetUserID(username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	var users []api.Autocomplete
	for i, user := range usernames.Results {
		if strings.EqualFold(user.Value, user.SearchTerm) {
			users = append(users, usernames.Results[i])
			break
		}
	}

	if len(users) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no users found"})
	}

	bin, err := json.Marshal(users)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	_ = cacheToUse.Set(key, &cache.Item{
		Blob:     bin,
		MimeType: echo.MIMEApplicationJSON,
	}, cache.Month)

	item, errFunc := cache.Retrieve(c, cacheToUse, cache.Fetch{
		URL:      fmt.Sprintf("https://jp.ib.metapix.net/usericons/small/%v", users[0].Icon),
		MimeType: cache.MimeTypeFromURL(users[0].Icon),
	})
	if errFunc != nil {
		return errFunc(c)
	}
	return c.Blob(http.StatusOK, item.MimeType, item.Blob)
}

func HandlePath(c echo.Context) error {
	path := c.Param("path")
	return ProxyHandler(path)(c)
}

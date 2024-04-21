package service

import (
	"encoding/json"
	"fmt"
	"github.com/ellypaws/inkbunny-app/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	units "github.com/labstack/gommon/bytes"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func RetrieveSubmission(c echo.Context, req api.SubmissionDetailsRequest) (api.SubmissionDetailsResponse, error) {
	var submissionDetails api.SubmissionDetailsResponse

	key := fmt.Sprintf("%s:inkbunny:submissions:%s?sid=%s", echo.MIMEApplicationJSON, req.SubmissionIDs, req.SID)
	cacheToUse := cache.SwitchCache(c)

	item, errFunc := cacheToUse.Get(key)
	if errFunc == nil {
		if err := json.Unmarshal(item.Blob, &submissionDetails); err == nil {
			c.Logger().Debugf("Cache hit for %s", key)
		} else {
			c.Logger().Errorf("error unmarshaling submission details: %v", err)
			return submissionDetails, err
		}
	} else {
		c.Logger().Infof("Cache miss for %s retrieving submission...", key)
		var err error
		submissionDetails, err = api.Credentials{Sid: req.SID}.SubmissionDetails(req)
		if err != nil {
			return submissionDetails, err
		}
		bin, err := json.Marshal(submissionDetails)
		if err != nil {
			c.Logger().Errorf("error marshaling submission details: %v", err)
			return submissionDetails, err
		}
		err = cacheToUse.Set(key, &cache.Item{
			Blob:     bin,
			MimeType: echo.MIMEApplicationJSON,
		}, cache.Week)
		if err != nil {
			c.Logger().Errorf("error caching submission details: %v", err)
		} else {
			c.Logger().Infof("Cached %s %s %dKiB", key, echo.MIMEApplicationJSON, len(bin)/units.KiB)
		}
	}

	return submissionDetails, nil
}

func RetrieveSearch(c echo.Context, request api.SubmissionSearchRequest) (api.SubmissionSearchResponse, error) {
	cacheToUse := cache.SwitchCache(c)

	if request.RID != "" {
		key := fmt.Sprintf("%s:inkbunny:search:%s:%s", echo.MIMEApplicationJSON, request.RID, request.Page)
		item, err := cacheToUse.Get(key)
		if err == nil {
			var response api.SubmissionSearchResponse
			if err := json.Unmarshal(item.Blob, &response); err == nil {
				c.Logger().Debugf("Cache hit for %s", key)
				return response, c.Blob(http.StatusOK, item.MimeType, item.Blob)
			}
		} else {
			c.Logger().Infof("Cache miss for %s retrieving search...", key)
		}
	}

	if !request.GetRID {
		c.Logger().Warn("GetRID was explicitly set to false, overriding to true...")
		request.GetRID = true
	}

	user := &api.Credentials{Sid: request.SID}
	request.SID = user.Sid
	searchResponse, err := user.SearchSubmissions(request)
	if err != nil {
		return searchResponse, c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}
	if len(searchResponse.Submissions) == 0 {
		return searchResponse, c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}

	ttl := 15 * time.Minute
	if searchResponse.RIDTTL != "" {
		var d time.Duration
		matches := regexp.MustCompile(`\d+[smhdwmy]`).FindAllString(strings.ReplaceAll(searchResponse.RIDTTL, " ", ""), -1)
		for _, match := range matches {
			i, err := strconv.Atoi(match[:len(match)-1])
			if err != nil {
				c.Logger().Errorf("error parsing RIDTTL: %v", err)
				continue
			}
			switch match[len(match)-1] {
			case 's':
				d += time.Second * time.Duration(i)
			case 'm':
				d += time.Minute * time.Duration(i)
			case 'h':
				d += time.Hour * time.Duration(i)
			case 'd':
				d += time.Hour * 24 * time.Duration(i)
			case 'w':
				d += time.Hour * 24 * 7 * time.Duration(i)
			case 'y':
				d += time.Hour * 24 * 365 * time.Duration(i)
			}
		}
		ttl = max(ttl, d)
	} else {
		c.Logger().Warn("RIDTTL was not set, using default 15 minutes")
	}

	if searchResponse.RID != "" {
		bin, err := json.Marshal(searchResponse)
		if err != nil {
			c.Logger().Errorf("error marshaling search response: %v", err)
			return searchResponse, c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		key := fmt.Sprintf("%s:inkbunny:search:%s:%s", echo.MIMEApplicationJSON, searchResponse.RID, request.Page)
		err = cacheToUse.Set(key, &cache.Item{
			Blob:     bin,
			MimeType: echo.MIMEApplicationJSON,
		}, ttl)
		if err != nil {
			c.Logger().Errorf("error caching search response: %v", err)
		} else {
			c.Logger().Infof("Cached %s %s %dKiB", key, echo.MIMEApplicationJSON, len(bin)/units.KiB)
		}
	} else {
		c.Logger().Warn("RID was not set, not caching search response")
	}

	return searchResponse, nil
}

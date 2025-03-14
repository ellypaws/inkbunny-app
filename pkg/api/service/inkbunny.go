package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	units "github.com/labstack/gommon/bytes"

	"github.com/ellypaws/inkbunny-app/pkg/api/cache"
	"github.com/ellypaws/inkbunny-app/pkg/crashy"
	"github.com/ellypaws/inkbunny-app/pkg/db"
)

// BatchRetrieveSubmission calls RetrieveSubmission in batches of 100, as that's the query limit.
// It uses a pool of workers to do this concurrently.
func BatchRetrieveSubmission(c echo.Context, req api.SubmissionDetailsRequest, missed []string) (api.SubmissionDetailsResponse, error) {
	var submissionDetails api.SubmissionDetailsResponse

	if len(missed) == 0 {
		return submissionDetails, crashy.ErrorResponse{ErrorString: "no submission IDs provided"}
	}

	const batchSize = 100
	if len(missed) <= batchSize {
		return RetrieveSubmission(c, req)
	}

	// calculate the number of jobs depending on the batch size
	var jobs = make(chan string, (len(missed)+batchSize-1)/batchSize)
	var responses = make(chan api.SubmissionDetailsResponse, len(missed))
	var errors = make(chan error, len(missed))

	work := func(id int, req api.SubmissionDetailsRequest, jobs <-chan string) {
		for submissionIDs := range jobs {
			c.Logger().Debugf("Worker %d retrieving description %s", id, submissionIDs)
			req.SubmissionIDs = submissionIDs
			details, err := RetrieveSubmission(c, req)
			if err != nil {
				errors <- err
				return
			}
			responses <- details
		}
	}

	const workers = 3
	for i := 0; i < workers; i++ {
		go work(i, req, jobs)
	}

	var jobCount int
	for i := 0; i < len(missed); i += batchSize {
		end := i + batchSize
		if end > len(missed) {
			end = len(missed)
		}
		jobs <- strings.Join(missed[i:end], ",")
		jobCount++
	}
	close(jobs)

	for i := 0; i < jobCount; i++ {
		select {
		case response := <-responses:
			submissionDetails.Sid = response.Sid
			submissionDetails.UserLocation = response.UserLocation
			submissionDetails.Submissions = append(submissionDetails.Submissions, response.Submissions...)
			submissionDetails.ResultsCount += response.ResultsCount
		case err := <-errors:
			return submissionDetails, err
		}
	}

	return submissionDetails, nil
}

func RetrieveSubmission(c echo.Context, req api.SubmissionDetailsRequest) (api.SubmissionDetailsResponse, error) {
	var submissionDetails api.SubmissionDetailsResponse

	key := fmt.Sprintf("%s:inkbunny:submissions:%s?sid=%s", echo.MIMEApplicationJSON, req.SubmissionIDs, db.Hash(req.SID))
	cacheToUse := cache.SwitchCache(c)

	if c.Request().Header.Get(echo.HeaderCacheControl) != "no-cache" {
		item, errFunc := cacheToUse.Get(key)
		if errFunc == nil {
			err := json.Unmarshal(item.Blob, &submissionDetails)
			if err == nil {
				c.Logger().Debugf("Cache hit for %s", key)
			} else {
				c.Logger().Errorf("error unmarshaling submission details: %v", err)
			}
			return submissionDetails, err
		}

		c.Logger().Infof("Cache miss for %s retrieving submission...", key)
	}

	var err error
	submissionDetails, err = api.Credentials{Sid: req.SID}.SubmissionDetails(req)
	if err != nil {
		return submissionDetails, err
	}
	slices.Reverse(submissionDetails.Submissions)

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

	return submissionDetails, nil
}

var shortDuration = regexp.MustCompile(`\d+[smhdwy]`)

func RetrieveSearch(c echo.Context, request api.SubmissionSearchRequest) (api.SubmissionSearchResponse, error) {
	cacheToUse := cache.SwitchCache(c)

	if c.Request().Header.Get(echo.HeaderCacheControl) == "no-cache" && request.RID != "" {
		c.Logger().Warn("Cache-Control is set to no-cache but RID is also set, bypassing cache...")
		request.RID = ""
	}

	if request.Page < 1 {
		c.Logger().Warnf("Page is set to %d, overriding to 1...", request.Page)
		request.Page = 1
	}

	if request.RID != "" {
		key := fmt.Sprintf("%s:inkbunny:search:%s:%d", echo.MIMEApplicationJSON, request.RID, request.Page)
		item, err := cacheToUse.Get(key)
		if err == nil {
			var response api.SubmissionSearchResponse
			if err := json.Unmarshal(item.Blob, &response); err != nil {
				c.Logger().Errorf("error unmarshaling search response: %v", err)
				return response, err
			}
			c.Logger().Debugf("Cache hit for %s", key)
			return response, nil
		} else {
			c.Logger().Infof("Cache miss for %s retrieving search...", key)
		}
	} else if !request.GetRID {
		c.Logger().Warn("GetRID was explicitly set to false but no RID is set, overriding to true...")
		request.GetRID = true
	}

	user := &api.Credentials{Sid: request.SID}
	request.SID = user.Sid
	searchResponse, err := user.SearchSubmissions(request)
	if err != nil {
		return searchResponse, crashy.ErrorResponse{ErrorString: "error searching submissions", Debug: err}
	}
	if len(searchResponse.Submissions) == 0 {
		return searchResponse, crashy.ErrorResponse{ErrorString: "no submissions found"}
	}

	ttl := 15 * time.Minute
	if searchResponse.RIDTTL != "" {
		var d time.Duration
		matches := shortDuration.FindAllString(strings.ReplaceAll(searchResponse.RIDTTL, " ", ""), -1)
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
			return searchResponse, crashy.ErrorResponse{ErrorString: "error marshaling search response", Debug: err}
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

func RetrieveUsers(c echo.Context, username string, exact bool) ([]api.Autocomplete, error) {
	key := fmt.Sprintf("%v:inkbunny:username_autosuggest:%v", echo.MIMEApplicationJSON, username)
	if exact {
		key = fmt.Sprintf("%v:inkbunny:username_autosuggest:exact:%v", echo.MIMEApplicationJSON, username)
	}

	cacheToUse := cache.SwitchCache(c)

	item, errFunc := cacheToUse.Get(key)
	if errFunc == nil {
		var users []api.Autocomplete
		if err := json.Unmarshal(item.Blob, &users); err != nil {
			return nil, err
		}

		if len(users) == 0 {
			return nil, crashy.ErrorResponse{ErrorString: "no users found"}
		}

		return users, nil
	}

	c.Logger().Infof("Cache miss for %s retrieving user...", key)

	usernames, err := api.GetUserID(username)
	if err != nil {
		return nil, err
	}

	if len(usernames.Results) == 0 {
		return nil, crashy.ErrorResponse{ErrorString: "no users found"}
	}

	var users = make([]api.Autocomplete, 0, len(usernames.Results))
	if exact {
		for i, user := range usernames.Results {
			if strings.EqualFold(user.Value, user.SearchTerm) {
				users = append(users, usernames.Results[i])
				break
			}
		}
	} else {
		users = usernames.Results
	}

	if len(users) == 0 {
		return nil, crashy.ErrorResponse{ErrorString: "no users found"}
	}

	bin, err := json.Marshal(users)
	if err != nil {
		c.Logger().Errorf("error marshaling user details: %v", err)
		return users, err
	}

	err = cacheToUse.Set(key, &cache.Item{
		Blob:     bin,
		MimeType: echo.MIMEApplicationJSON,
	}, cache.Week)
	if err != nil {
		c.Logger().Errorf("error caching user details: %v", err)
	} else {
		c.Logger().Infof("Cached %s %s %dKiB", key, echo.MIMEApplicationJSON, len(bin)/units.KiB)
	}

	return users, nil
}

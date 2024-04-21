package service

import (
	"encoding/json"
	"fmt"
	"github.com/ellypaws/inkbunny-app/api/cache"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	units "github.com/labstack/gommon/bytes"
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

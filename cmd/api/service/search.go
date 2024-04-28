package service

import (
	"fmt"
	"github.com/ellypaws/inkbunny-app/cmd/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const ReviewSearchFormat = "%s:review:%s:search:%s:%d?%s"

func RetrieveReviewSearch(c echo.Context, sid string, output string, query url.Values, cacheToUse cache.Cache) (*api.SubmissionSearchResponse, func(echo.Context) error) {
	var request = api.SubmissionSearchRequest{
		Text:               "ai_generated",
		SubmissionsPerPage: 10,
		Random:             true,
		GetRID:             true,
		SubmissionIDsOnly:  true,
		Type:               api.SubmissionTypes{api.SubmissionTypePicturePinup},
	}
	var bind = struct {
		*api.SubmissionSearchRequest
		Types *string `json:"types,omitempty" query:"types"`
	}{
		SubmissionSearchRequest: &request,
	}
	err := c.Bind(&bind)
	if err != nil {
		return nil, cache.ErrFunc(http.StatusBadRequest, err)
	}

	request.SID = sid
	if c.Request().Header.Get(echo.HeaderCacheControl) == "no-cache" {
		request.RID = ""
	}

	if request.Page < 1 {
		c.Logger().Warnf("Page is set to %d, overriding to 1...", request.Page)
		request.Page = 1
	}

	if request.RID != "" {
		searchReviewKey := fmt.Sprintf(
			ReviewSearchFormat,
			echo.MIMEApplicationJSON,
			output,
			request.RID,
			request.Page,
			query.Encode(),
		)
		item, err := cacheToUse.Get(searchReviewKey)
		if err == nil {
			c.Logger().Infof("Cache hit for %s", searchReviewKey)
			return nil, func(c echo.Context) error { return c.Blob(http.StatusOK, item.MimeType, item.Blob) }
		}
	}

	if bind.Types != nil {
		*bind.Types = strings.Trim(*bind.Types, "[]")
		*bind.Types = strings.ReplaceAll(*bind.Types, `"`, "")
		for _, t := range strings.Split(*bind.Types, ",") {
			i, err := strconv.Atoi(t)
			if err != nil {
				return nil, cache.ErrFunc(http.StatusBadRequest, crashy.ErrorResponse{
					ErrorString: fmt.Sprintf("invalid type: %s", t),
					Debug:       err,
				})
			}
			request.Type = append(request.Type, api.SubmissionType(i))
		}
	}

	searchResponse, err := RetrieveSearch(c, request)
	if err != nil {
		return nil, cache.ErrFunc(http.StatusInternalServerError, err)
	}

	return &searchResponse, nil
}

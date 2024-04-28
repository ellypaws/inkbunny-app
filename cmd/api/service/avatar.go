package service

import (
	"fmt"
	"github.com/ellypaws/inkbunny-app/cmd/api/cache"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
)

func RetrieveAvatar(c echo.Context, cacheToUse cache.Cache, user api.Autocomplete) (*cache.Item, func(c echo.Context) error) {
	iconURL := fmt.Sprintf("https://jp.ib.metapix.net/usericons/small/%v", user.Icon)
	mimeType := cache.MimeTypeFromURL(user.Icon)
	imageKey := fmt.Sprintf("%v:%v", mimeType, iconURL)
	if user.Icon == "" {
		iconURL = "https://jp.ib.metapix.net/images80/usericons/small/noicon.png"
		mimeType = "image/png"
		imageKey = fmt.Sprintf("%v:%v", mimeType, iconURL)
	}

	item, errFunc := cache.Retrieve(c, cacheToUse, cache.Fetch{
		Key:      imageKey,
		URL:      iconURL,
		MimeType: mimeType,
	})
	if errFunc != nil {
		return nil, errFunc
	}
	return item, nil
}

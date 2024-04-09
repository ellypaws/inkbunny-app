package caption

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ellypaws/inkbunny-app/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	e "github.com/ellypaws/inkbunny-sd/entities"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
	"github.com/labstack/echo/v4"
	"strings"
	"sync"
	"time"
)

var defaultThreshold = 0.3

var defaultTaggerRequest = e.TaggerRequest{
	Model:     e.TaggerZ3DE621Convnext,
	Threshold: &defaultThreshold,
}

func ProcessCaption(c echo.Context, wg *sync.WaitGroup, sub *db.Submission, i int, host *sd.Host) {
	defer wg.Done()
	f := &sub.Files[i].File
	if !strings.HasPrefix(f.MimeType, "image") {
		return
	}

	var cacheToUse cache.Cache
	redis, ok := c.Get("redis").(*cache.Redis)
	if ok {
		cacheToUse = redis
	} else {
		cacheToUse = cache.GetLocalCache(c)
	}

	key := fmt.Sprintf("%s:%s", echo.MIMEApplicationJSON, f.FileURLScreen)

	item, err := cacheToUse.Get(key)
	if err == nil {
		var result *e.CaptionEnum
		err := json.Unmarshal(item.Blob, &result)
		if err != nil {
			c.Logger().Errorf("error unmarshaling caption: %v", err)
			return
		}
		c.Logger().Infof("Cache hit for %s", key)
		sub.Files[i].Caption = result
		return
	}

	if !host.Alive() {
		return
	}

	c.Logger().Debugf("Cache miss for %s retrieving image...", key)

	item, errorFunc := cache.Retrieve(c, cacheToUse, fmt.Sprintf("%v:%v", f.MimeType, f.FileURLScreen), f.FileURLScreen)
	if errorFunc != nil {
		return
	}
	req := defaultTaggerRequest

	base64String := base64.StdEncoding.EncodeToString(item.Blob)
	req.Image = &base64String
	*req.Threshold = 0.7

	c.Logger().Infof("Interrogating captions for %v", f.FileURLScreen)
	t, err := host.Interrogate(&req)
	if err != nil {
		c.Logger().Errorf("error processing captions for %v: %v", f.FileURLScreen, err)
		return
	}

	c.Logger().Debugf("finished captions for %v", f.FileURLScreen)

	sub.Metadata.HumanConfidence = max(sub.Metadata.HumanConfidence, t.HumanPercent())
	if t.HumanPercent() > 0.5 {
		sub.Metadata.DetectedHuman = true
	}

	blob, err := json.Marshal(t.Caption)
	if err != nil {
		c.Logger().Errorf("error marshaling caption: %v", err)
		return
	}

	err = cacheToUse.Set(key, &cache.Item{
		Blob:       blob,
		LastAccess: time.Now().UTC(),
		MimeType:   echo.MIMEApplicationJSON,
	})
	if err != nil {
		c.Logger().Errorf("error caching caption: %v", err)
	}

	sub.Files[i].Caption = &t.Caption
}

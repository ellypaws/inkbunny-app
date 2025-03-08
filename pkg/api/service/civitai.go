package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/bytes"

	"github.com/ellypaws/inkbunny-app/pkg/api/cache"
	"github.com/ellypaws/inkbunny-app/pkg/api/civitai"
	"github.com/ellypaws/inkbunny-app/pkg/crashy"
	"github.com/ellypaws/inkbunny-app/pkg/db"
	"github.com/ellypaws/inkbunny-sd/stable_diffusion"
)

func QueryCivitAI(c echo.Context, cacheToUse cache.Cache, hash string) (db.ModelHashes, *civitai.CivitAIModel, error) {
	key := fmt.Sprintf("%s:civitai:%s", echo.MIMEApplicationJSON, hash)
	civ := c.QueryParam("civitai") == "true"

	var model *civitai.CivitAIModel

	item, err := cacheToUse.Get(key)
	if err == nil {
		c.Logger().Debugf("Cache hit for %s", key)
		if err := json.Unmarshal(item.Blob, &model); err != nil {
			return nil, nil, err
		}
		if civ {
			return nil, model, nil
		}
	}

	if model == nil {
		model, err = civitai.DefaultHost.GetByHash(hash)
		if err != nil {
			c.Logger().Errorf("model %s not found in CivitAI", hash)
			return nil, nil, crashy.ErrorResponse{ErrorString: "model not found"}
		}

		// TODO: Not yet implemented. This is where we download the model if found in CivitAI.
		err = sd.DownloadModel(hash)

		bin, err := json.Marshal(model)
		if err != nil {
			return nil, nil, err
		}

		if err = cacheToUse.Set(key, &cache.Item{
			Blob:     bin,
			MimeType: echo.MIMEApplicationJSON,
		}, cache.Month); err != nil {
			c.Logger().Errorf("error caching CivitAI model %s: %v", hash, err)
		} else {
			c.Logger().Infof("Cached %s %s %dKiB", key, echo.MIMEApplicationJSON, len(bin)/bytes.KiB)
		}
	}

	var name string
	for _, file := range model.Files {
		var hashToCheck string
		switch len(hash) {
		case 10:
			hashToCheck = file.Hashes.AutoV2
		case 12:
			hashToCheck = file.Hashes.AutoV3
		}
		if strings.EqualFold(hash, hashToCheck) {
			name = file.Name
			if !file.Primary {
				c.Logger().Warnf("model %s has a non-primary file: %s", model.Name, file.Name)
			}
			c.Logger().Infof("download url is %s", file.DownloadURL)
			break
		}
	}

	if name == "" {
		msg := fmt.Sprintf("hash %s not found in model %s", hash, model.Name)
		c.Logger().Error(msg)
		return nil, nil, crashy.ErrorResponse{ErrorString: msg, Debug: model}
	}

	return db.ModelHashes{hash: []string{model.Name, name}}, model, err
}

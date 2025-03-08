package service

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	units "github.com/labstack/gommon/bytes"
	logger "github.com/labstack/gommon/log"

	"github.com/ellypaws/inkbunny-app/pkg/api/cache"
	"github.com/ellypaws/inkbunny-app/pkg/crashy"
	"github.com/ellypaws/inkbunny-app/pkg/db"
	"github.com/ellypaws/inkbunny-sd/entities"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
)

func Inference(c echo.Context, object map[string]entities.TextToImageRequest, host *sd.Host, database *db.Sqlite) (map[string]entities.TextToImageResponse, error) {
	var responses map[string]entities.TextToImageResponse
	cacheToUse := cache.SwitchCache(c)
	for key, request := range object {
		if request.Prompt == "" {
			c.Logger().Warnf("prompt is empty for %s", key)
		}
		if request.OverrideSettings.SDModelCheckpoint == nil {
			c.Logger().Warnf("model checkpoint is empty for %s", key)
		}

		key = fmt.Sprintf("%s:generation:%s", echo.MIMEApplicationJSON, key)
		regenerate := c.QueryParam("regenerate") == "true"

		var item *cache.Item
		if !regenerate {
			var err error
			item, err = cacheToUse.Get(key)
			if err == nil {
				response, err := entities.UnmarshalTextToImageResponse(item.Blob)
				if err != nil {
					c.Logger().Errorf("error unmarshaling response: %v", err)
					return nil, c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
				}
				c.Logger().Debugf("Cache hit for %s", key)

				if c.QueryParam("image") == "true" {
					if len(response.Images) == 0 {
						return nil, c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no images were generated"})
					}
					bin, err := base64.StdEncoding.DecodeString(response.Images[0])
					if err != nil {
						return nil, c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
					}
					return nil, c.Blob(http.StatusOK, "image/png", bin)
				}
				if responses == nil {
					responses = make(map[string]entities.TextToImageResponse)
				}
				responses[key] = response
				continue
			}
		} else {
			c.Logger().Infof("Regenerating %s", key)
		}

		for hash, name := range request.LoraHashes {
			match, err := QueryHost(c, cacheToUse, host, database, hash)
			if err != nil {
				c.Logger().Warnf("error querying host: %v", err)
			}
			if match == nil {
				c.Logger().Warnf("lora %s isn't downloaded: %s", name, hash)
				level := c.Logger().Level()
				c.Logger().SetLevel(logger.INFO)
				_, _, _ = QueryCivitAI(c, cacheToUse, hash)
				c.Logger().SetLevel(level)
			} else {
				c.Logger().Debugf("Found lora %s: %s", name, hash)
			}
		}

		if request.OverrideSettings.SDModelCheckpoint != nil {
			checkpoints, err := host.GetCheckpoints()
			if err != nil {
				return nil, c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
			}

			var found bool
			for _, checkpoint := range checkpoints {
				if checkpoint.Title == *request.OverrideSettings.SDModelCheckpoint {
					found = true
				}
			}
			if !found {
				c.Logger().Warnf("Model checkpoint %s not found", *request.OverrideSettings.SDModelCheckpoint)
				level := c.Logger().Level()
				c.Logger().SetLevel(logger.INFO)
				_, _, err := QueryCivitAI(c, cacheToUse, request.OverrideSettings.SDCheckpointHash[:10])
				if err != nil {
					return nil, c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
				}
				c.Logger().SetLevel(level)
				// return c.JSON(http.StatusNotFound, civ)
			} else {
				c.Logger().Infof("Found model checkpoint %s", *request.OverrideSettings.SDModelCheckpoint)
			}
		}

		c.Logger().Infof("Generating %s...", key)
		response, err := host.TextToImageRequest(&request)
		if err != nil {
			return nil, c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		generation := entities.TextToImageResponse{
			Images:     response.Images,
			Seeds:      response.Seeds,
			Subseeds:   response.Subseeds,
			Parameters: response.Parameters,
			Info:       response.Info,
		}

		if len(generation.Images) == 0 {
			return nil, c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no images were generated"})
		}

		c.Logger().Infof("Finished %s", key)
		bin, err := generation.Marshal()
		if err != nil {
			return nil, c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		err = cacheToUse.Set(key, &cache.Item{
			Blob:     bin,
			MimeType: echo.MIMEApplicationJSON,
		}, cache.Week)
		if err != nil {
			c.Logger().Errorf("error caching generation: %v", err)
		} else {
			c.Logger().Infof("Cached %s %dKiB", key, len(bin)/units.KiB)
		}

		if i := c.QueryParam("image"); i == "true" {
			if len(generation.Images) == 0 {
				return nil, c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no images were generated"})
			}
			bin, err := base64.StdEncoding.DecodeString(response.Images[0])
			if err != nil {
				return nil, c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
			}
			return nil, c.Blob(http.StatusOK, "image/png", bin)
		}

		if responses == nil {
			responses = make(map[string]entities.TextToImageResponse)
		}
		responses[key] = generation
	}

	if len(responses) == 0 {
		return nil, c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no responses were generated"})
	}

	return responses, nil
}

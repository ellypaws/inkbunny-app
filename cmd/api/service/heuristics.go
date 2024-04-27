package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ellypaws/inkbunny-app/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/utils"
	"github.com/labstack/echo/v4"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

func RetrieveParams(c echo.Context, wg *sync.WaitGroup, sub *db.Submission, cache cache.Cache, artists []db.Artist) {
	defer wg.Done()

	key := fmt.Sprintf("%s:parameters:%d", echo.MIMEApplicationJSON, sub.ID)
	if c.Request().Header.Get(echo.HeaderCacheControl) != "no-cache" {
		item, err := cache.Get(key)
		if err == nil {
			var metadata db.Metadata
			err := json.Unmarshal(item.Blob, &metadata)
			if err != nil {
				c.Logger().Errorf("error unmarshaling params: %v", err)
				return
			}
			c.Logger().Debugf("Cache hit for %s", key)
			sub.Metadata.Objects = metadata.Objects
			sub.Metadata.Params = metadata.Params
			return
		}

		c.Logger().Debugf("Cache miss for %s retrieving params...", key)
	}

	// TODO: cache.Set after processing
	processParams(c, sub, cache, artists)
}

func processParams(c echo.Context, sub *db.Submission, cacheToUse cache.Cache, artists []db.Artist) {
	if sub.Metadata.Params != nil {
		return
	}

	var textFile *db.File

	for i, f := range sub.Files {
		switch f.File.MimeType {
		case echo.MIMEApplicationJSON:
			if strings.Contains(f.File.FileName, "plugin") {
				continue
			}
			textFile = &sub.Files[i]
			if strings.Contains(f.File.FileName, "workflow") {
				break
			}
		case echo.MIMETextPlain:
			textFile = &sub.Files[i]
			break
		}
	}

	defer processObjectMetadata(sub, artists)

	if textFile == nil {
		processDescriptionHeuristics(c, sub)
		return
	}

	b, errFunc := cache.Retrieve(c, cacheToUse, cache.Fetch{
		Key:      fmt.Sprintf("%s:%s", textFile.File.MimeType, textFile.File.FileURLFull),
		URL:      textFile.File.FileURLFull,
		MimeType: textFile.File.MimeType,
	})
	if errFunc != nil {
		return
	}

	if b.MimeType == echo.MIMEApplicationJSON {
		jsonHeuristics(c, sub, b, textFile)
		return
	}

	if parameterHeuristics(c, sub, textFile, b) {
		return
	}

	if len(sub.Metadata.Objects) == 0 {
		processDescriptionHeuristics(c, sub)
		return
	}
}

// deferred call to set metadata flags after processing objects
func processObjectMetadata(submission *db.Submission, artists []db.Artist) {
	submission.Metadata.MissingPrompt = true
	submission.Metadata.MissingModel = true

	for _, obj := range submission.Metadata.Objects {
		submission.Metadata.AISubmission = true
		meta := strings.ToLower(obj.Prompt + obj.NegativePrompt)
		for _, artist := range artists {
			re, err := regexp.Compile(fmt.Sprintf(`\b%s\b`, strings.ToLower(artist.Username)))
			if err != nil {
				continue
			}
			if re.MatchString(meta) {
				submission.Metadata.ArtistUsed = append(submission.Metadata.ArtistUsed, artist)
			}
		}

		privateTools := []string{
			"midjourney",
			"novelai",
		}

		for _, tool := range privateTools {
			if strings.Contains(meta, tool) {
				submission.Metadata.PrivateTool = true
				submission.Metadata.Generator = tool
				break
			}
		}

		if obj.Prompt != "" {
			submission.Metadata.MissingPrompt = false
		}

		if obj.OverrideSettings.SDModelCheckpoint != nil || obj.OverrideSettings.SDCheckpointHash != "" {
			submission.Metadata.MissingModel = false
		}
	}
}

func jsonHeuristics(c echo.Context, sub *db.Submission, b *cache.Item, textFile *db.File) {
	comfyUI, err := entities.UnmarshalComfyUIBasic(b.Blob)
	if err != nil {
		c.Logger().Errorf("error parsing comfy ui %s: %s", textFile.File.FileURLFull, err)
	}
	if err == nil && len(comfyUI.Nodes) > 0 {
		c.Logger().Debugf("comfy ui found for %s", sub.URL)
		sub.Metadata.Objects = map[string]entities.TextToImageRequest{
			textFile.File.FileName: *comfyUI.Convert(),
		}
		sub.Metadata.Params = &utils.Params{
			textFile.File.FileName: utils.PNGChunk{
				"comfy_ui": string(b.Blob),
			},
		}
		sub.Metadata.Generator = "comfy_ui"
		return
	}

	easyDiffusion, err := entities.UnmarshalEasyDiffusion(b.Blob)
	if err != nil {
		c.Logger().Errorf("error parsing easy diffusion %s: %s", textFile.File.FileURLFull, err)
	}
	if err == nil && !reflect.DeepEqual(easyDiffusion, entities.EasyDiffusion{}) {
		c.Logger().Debugf("easy diffusion found for %s", sub.URL)
		sub.Metadata.Objects = map[string]entities.TextToImageRequest{
			textFile.File.FileName: *easyDiffusion.Convert(),
		}
		sub.Metadata.Params = &utils.Params{
			textFile.File.FileName: utils.PNGChunk{
				"easy_diffusion": string(b.Blob),
			},
		}
		sub.Metadata.Generator = "easy_diffusion"
		return
	}

	c.Logger().Warnf("could not parse json %s for %s", textFile.File.FileURLFull, sub.URL)
	return
}

// Because some artists already have standardized txt files, opt to split each file separately
func parameterHeuristics(c echo.Context, sub *db.Submission, textFile *db.File, b *cache.Item) bool {
	var params utils.Params
	var err error
	f := &textFile.File
	c.Logger().Debugf("processing params for %s", f.FileName)
	switch sub.UserID {
	case utils.IDAutoSnep:
		params, err = utils.AutoSnep(utils.WithBytes(b.Blob))
	case utils.IDDruge:
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseDruge())
	case utils.IDAIBean:
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseAIBean())
	case utils.IDArtieDragon:
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseArtie())
	case 1125540:
		params, err = utils.Common(
			utils.WithBytes(b.Blob),
			utils.WithFilename("picker52578_"),
			utils.WithKeyCondition(func(line string) bool { return strings.HasPrefix(line, "File Name") }))
	case utils.IDFairyGarden:
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseFairyGarden())
	case utils.IDCirn0:
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseCirn0())
	case utils.IDHornybunny:
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseHornybunny())
	default:
		params, err = utils.Common(
			// prepend "photo 1" to the input in case it's missing
			utils.WithBytes(bytes.Join([][]byte{[]byte(f.FileName), b.Blob}, []byte("\n"))),
			utils.WithKeyCondition(func(line string) bool { return strings.HasPrefix(line, f.FileName) }))
	}
	if err != nil {
		c.Logger().Errorf("error processing params for %s: %s", f.FileName, err)
		return true
	}
	if len(params) > 0 {
		c.Logger().Debugf("finished params for %s", f.FileName)
		sub.Metadata.Params = &params
		paramsToObject(c, sub)
	}
	return false
}

func paramsToObject(c echo.Context, sub *db.Submission) {
	if sub.Metadata.Objects != nil {
		return
	}

	var wg sync.WaitGroup
	var mutex sync.Mutex
	for fileName, params := range *sub.Metadata.Params {
		if p, ok := params[utils.Parameters]; ok {
			c.Logger().Debugf("processing heuristics for %v", fileName)
			wg.Add(1)
			go func(name string, content string) {
				defer wg.Done()
				heuristics, err := utils.ParameterHeuristics(content)
				if err != nil {
					c.Logger().Errorf("error processing heuristics for %v: %v", name, err)
					return
				}
				if sub.Metadata.Objects == nil {
					sub.Metadata.Objects = make(map[string]entities.TextToImageRequest)
				}
				mutex.Lock()
				sub.Metadata.Objects[name] = heuristics
				mutex.Unlock()
			}(fileName, p)
		}
	}
	wg.Wait()
}

func processDescriptionHeuristics(c echo.Context, sub *db.Submission) {
	c.Logger().Debugf("processing description heuristics for %v", sub.URL)
	heuristics, err := utils.DescriptionHeuristics(sub.Description)
	if err != nil {
		c.Logger().Errorf("error processing description heuristics for %v: %v", sub.URL, err)
		return
	}
	if reflect.DeepEqual(heuristics, entities.TextToImageRequest{}) {
		c.Logger().Debugf("no heuristics found for %v", sub.URL)
		return
	}
	sub.Metadata.Objects = map[string]entities.TextToImageRequest{sub.Title: heuristics}
}

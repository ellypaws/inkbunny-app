package service

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	units "github.com/labstack/gommon/bytes"
	"github.com/lu4p/cat/rtftxt"

	"github.com/ellypaws/inkbunny-app/pkg/api/cache"
	"github.com/ellypaws/inkbunny-app/pkg/db"

	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/entities/comfyui"
	"github.com/ellypaws/inkbunny-sd/utils"
)

func RetrieveParams(c echo.Context, wg *sync.WaitGroup, sub *db.Submission, cacheToUse cache.Cache, artists []db.Artist) {
	defer wg.Done()

	key := fmt.Sprintf("%s:parameters:%d", echo.MIMEApplicationJSON, sub.ID)
	if c.Request().Header.Get(echo.HeaderCacheControl) != "no-cache" {
		item, err := cacheToUse.Get(key)
		if err == nil {
			var metadata db.Metadata
			err := json.Unmarshal(item.Blob, &metadata)
			if err != nil {
				c.Logger().Errorf("error unmarshaling params: %v", err)
				return
			}
			c.Logger().Debugf("Cache hit for %s", key)

			sub.Metadata.Generated = metadata.Generated
			sub.Metadata.Assisted = metadata.Assisted
			sub.Metadata.Img2Img = metadata.Img2Img
			sub.Metadata.HasJSON = metadata.HasJSON
			sub.Metadata.HasTxt = metadata.HasTxt
			sub.Metadata.StableDiffusion = metadata.StableDiffusion
			sub.Metadata.ComfyUI = metadata.ComfyUI
			sub.Metadata.MultipleImages = metadata.MultipleImages
			sub.Metadata.TaggedHuman = metadata.TaggedHuman

			sub.Metadata.AITitle = metadata.AITitle
			sub.Metadata.AIDescription = metadata.AIDescription
			sub.Metadata.AIKeywords = metadata.AIKeywords
			sub.Metadata.AIAccount = metadata.AIAccount
			sub.Metadata.AISubmission = metadata.AISubmission
			sub.Metadata.MissingPrompt = metadata.MissingPrompt
			sub.Metadata.MissingModel = metadata.MissingModel
			sub.Metadata.MissingTags = metadata.MissingTags
			sub.Metadata.ArtistUsed = metadata.ArtistUsed
			sub.Metadata.PrivateModel = metadata.PrivateModel
			sub.Metadata.PrivateLora = metadata.PrivateLora
			sub.Metadata.PrivateTool = metadata.PrivateTool
			sub.Metadata.SoldArt = metadata.SoldArt
			sub.Metadata.Generator = metadata.Generator

			sub.Metadata.Params = metadata.Params
			sub.Metadata.Objects = metadata.Objects

			return
		}

		c.Logger().Debugf("Cache miss for %s retrieving params...", key)
	}

	processParams(c, sub, cacheToUse)
	processObjectMetadata(sub, artists)
	if sub.Metadata.Objects != nil || sub.Metadata.Params != nil {
		bin, err := json.Marshal(sub.Metadata)
		if err != nil {
			c.Logger().Errorf("error marshaling params: %v", err)
			return
		}

		err = cacheToUse.Set(key, &cache.Item{
			Blob:     bin,
			MimeType: echo.MIMEApplicationJSON,
		}, cache.Week)
		if err != nil {
			c.Logger().Errorf("error caching params: %v", err)
		} else {
			c.Logger().Infof("Cached %s %dKiB", key, len(bin)/units.KiB)
		}
	}
}

const MIMETextRTF = "text/rtf"

func processParams(c echo.Context, sub *db.Submission, cacheToUse cache.Cache) {
	if sub.Metadata.Params != nil {
		return
	}

	var textFiles []*db.File

	for i, f := range sub.Files {
		switch f.File.MimeType {
		case echo.MIMEApplicationJSON:
			if strings.Contains(f.File.FileName, "plugin") {
				continue
			}
			textFiles = append(textFiles, &sub.Files[i])
		case echo.MIMETextPlain, MIMETextRTF:
			textFiles = append(textFiles, &sub.Files[i])
		}
	}

	if len(textFiles) == 0 {
		processDescriptionHeuristics(c, sub)
		return
	}

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	for _, textFile := range textFiles {
		wg.Add(1)
		go func(textFile *db.File) {
			defer wg.Done()
			// TODO: shouldSave won't run if we've retrieved cached data
			c.Set("shouldSave", c.QueryParam("output") == OutputReport || c.QueryParam("output") == OutputReportIDs)
			threeMonths := 3 * cache.Month
			b, errFunc := cache.Retrieve(c, cacheToUse, cache.Fetch{
				Key:      fmt.Sprintf("%s:%s", textFile.File.MimeType, textFile.File.FileURLFull),
				URL:      textFile.File.FileURLFull,
				MimeType: textFile.File.MimeType,
				Duration: &threeMonths,
			})
			if errFunc != nil {
				c.Logger().Errorf("error fetching %s: (%s)", textFile.File.FileURLFull, sub.URL)
				return
			}

			if b.MimeType == MIMETextRTF {
				plain, err := rtftxt.Text(bytes.NewReader(b.Blob))
				if err != nil {
					c.Logger().Errorf("error parsing rtf %s: %s", textFile.File.FileURLFull, err)
					return
				}
				b.Blob = plain.Bytes()
			}

			if b.MimeType == echo.MIMEApplicationJSON || (bytes.HasPrefix(b.Blob, []byte("{")) && bytes.HasSuffix(b.Blob, []byte("}"))) {
				if jsonHeuristics(c, sub, b, textFile, &mu) {
					return
				}
			}

			if err := parameterHeuristics(c, sub, textFile, b, &mu); err != nil {
				c.Logger().Errorf("error processing params for %s: %v", textFile.File.FileName, err)
				return
			}
		}(textFile)
	}
	wg.Wait()

	if len(sub.Metadata.Objects) == 0 {
		processDescriptionHeuristics(c, sub)
		return
	}
}

var additionalArtists = regexp.MustCompile(`(?im)[\[({<|:,]\s*by ([^:,\r\n\])}>]+)|^by ([^:,\r\n\])}>]+)`)

// deferred call to set metadata flags after processing objects
func processObjectMetadata(submission *db.Submission, artists []db.Artist) {
	submission.Metadata.MissingPrompt = true
	submission.Metadata.MissingModel = true

	var sizes [2]int
	for _, f := range submission.Files {
		if f.File.FullSizeX == 0 || f.File.FullSizeY == 0 {
			continue
		}
		sizes[0] = max(sizes[0], int(f.File.FullSizeX))
		sizes[1] = max(sizes[1], int(f.File.FullSizeY))
		break
	}
	for _, obj := range submission.Metadata.Objects {
		submission.Metadata.AISubmission = true
		for _, artist := range artists {
			re, err := regexp.Compile(fmt.Sprintf(`(?i)\b%s\b`, strings.ToLower(artist.Username)))
			if err != nil {
				continue
			}
			if re.MatchString(obj.Prompt) {
				submission.Metadata.ArtistUsed = append(submission.Metadata.ArtistUsed, artist)
			}
		}

		additionalArtists := additionalArtists.FindAllStringSubmatch(obj.Prompt, -1)
		for _, match := range additionalArtists {
			for _, artist := range strings.Split(strings.Join(match[1:], ""), "|") {
				if !slices.ContainsFunc(submission.Metadata.ArtistUsed, func(stored db.Artist) bool {
					return strings.EqualFold(stored.Username, artist)
				}) {
					submission.Metadata.ArtistUsed = append(submission.Metadata.ArtistUsed, db.Artist{Username: artist})
				}
			}
		}

		if tool := PrivateTools.FindString(obj.Prompt); tool != "" {
			submission.Metadata.PrivateTool = true
			submission.Metadata.Generator = tool
		}

		if obj.Prompt != "" {
			submission.Metadata.MissingPrompt = false
		}

		if obj.OverrideSettings.SDModelCheckpoint != nil || obj.OverrideSettings.SDCheckpointHash != "" {
			submission.Metadata.MissingModel = false
		}

		if obj.Width == 0 || obj.Height == 0 {
			obj.Width = sizes[0]
			obj.Height = sizes[1]
		}
	}
}

func jsonHeuristics(c echo.Context, sub *db.Submission, b *cache.Item, textFile *db.File, mu *sync.Mutex) bool {
	b.Blob = bytes.ReplaceAll(b.Blob, []byte("NaN"), []byte("null"))
	best := preferBest(c, b.Blob, map[string]Converter{
		"comfy_ui":       &comfyui.Basic{},
		"comfy_ui_api":   &comfyui.Api{},
		"invoke_ai":      &entities.InvokeAI{},
		"easy_diffusion": &entities.EasyDiffusion{},
	})
	if best == nil {
		c.Logger().Warnf("no suitable type found for %s", sub.URL)
	} else {
		c.Logger().Debugf("%s found for %s", best.label, sub.URL)
		mu.Lock()
		insertOrInitalize(&sub.Metadata.Objects, map[string]entities.TextToImageRequest{
			textFile.File.FileName: *best.TextToImageRequest,
		})
		insertOrInitalize(&sub.Metadata.Params, utils.Params{
			textFile.File.FileName: utils.PNGChunk{
				best.label: string(b.Blob),
			},
		})
		mu.Unlock()
		sub.Metadata.Generator = best.label
		return true
	}

	if sub.UserID == 1247248 {
		cubFestAI, err := comfyui.UnmarshalCubFestAIDate(b.Blob)
		if err != nil {
			c.Logger().Warnf("error parsing comfy ui (CubFestAI) %s: %s", textFile.File.FileURLFull, err)
		}
		if err == nil && !reflect.DeepEqual(cubFestAI, comfyui.CubFestAITime{}) {
			c.Logger().Debugf("comfy ui cub fest ai found for %s", sub.URL)
			var objects = make(map[string]entities.TextToImageRequest)
			for key, value := range cubFestAI {
				objects[key] = value.Convert()
			}
			mu.Lock()
			insertOrInitalize(&sub.Metadata.Objects, objects)
			insertOrInitalize(&sub.Metadata.Params, utils.Params{
				textFile.File.FileName: utils.PNGChunk{
					"comfy_ui": string(b.Blob),
				},
			})
			mu.Unlock()
			sub.Metadata.Generator = "comfy_ui"
			return true
		}
	}

	c.Logger().Errorf("could not parse json %s for %s", textFile.File.FileURLFull, sub.URL)
	return false
}

type Converter interface {
	Convert() *entities.TextToImageRequest
}

type converted struct {
	label string
	*entities.TextToImageRequest
	err error
}

// preferBest takes a blob and a map of types, and returns the best match.
// It sorts the objects by prompt, seed, model, width, and height.
// Call preferBest with empty Converter types.
func preferBest(c echo.Context, blob []byte, types map[string]Converter) *converted {
	if len(types) == 0 {
		return nil
	}
	var (
		object  Converter
		objects []converted
		err     error
	)
	for label, zero := range types {
		switch zero.(type) {
		case *comfyui.Basic:
			object, err = unmarshal[*comfyui.Basic](blob)
		case *comfyui.Api:
			object, err = unmarshal[*comfyui.Api](blob)
		case *entities.EasyDiffusion:
			object, err = unmarshal[*entities.EasyDiffusion](blob)
		case *entities.InvokeAI:
			object, err = unmarshal[*entities.InvokeAI](blob)
		default:
			continue
		}
		if err != nil && !errors.Is(err, comfyui.ErrInvalidNode) {
			c.Logger().Warnf("error parsing %s: %v", label, err)
			continue
		}
		switch object := object.(type) {
		case *comfyui.Basic:
			if len(object.Nodes) == 0 {
				continue
			}
			var e comfyui.NodeErrors
			if errors.As(err, &e) {
				c.Logger().Warnf("parsed %s with some errors. errors/ok: %d/%d", label, e.Len(), len(object.Nodes))
			}
		case *comfyui.Api:
			if len(*object) == 0 {
				continue
			}
			var e comfyui.NodeErrors
			if errors.As(err, &e) {
				c.Logger().Warnf("parsed %s with some errors. errors/ok: %d/%d", label, e.Len(), len(*object))
			}
		default:
			if reflect.DeepEqual(object, zero) {
				continue
			}
		}
		objects = append(objects, converted{label, object.Convert(), err})
	}
	if len(objects) == 0 {
		return nil
	}
	if len(objects) == 1 {
		return &objects[0]
	}
	slices.SortFunc(objects, func(a, b converted) int {
		return cmp.Or(
			-cmp.Compare(a.Prompt, b.Prompt),
			-cmp.Compare(a.Seed, b.Seed),
			-comparePointer(a.OverrideSettings.SDModelCheckpoint, b.OverrideSettings.SDModelCheckpoint),
			-cmp.Compare(a.Width, b.Width),
			-cmp.Compare(a.Height, b.Height),
		)
	})
	return &objects[0]
}

func unmarshal[T any](data []byte) (T, error) {
	var zero T
	err := json.Unmarshal(data, &zero)
	return zero, err
}

func comparePointer[T cmp.Ordered](a, b *T) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	return cmp.Compare(*a, *b)
}

func insertOrInitalize[M interface{ ~map[K]V }, K comparable, V any](m *M, v M) bool {
	if m == nil {
		return false
	}
	if *m == nil {
		*m = v
		return v != nil
	}
	maps.Copy(*m, v)
	return true
}

// Because some artists already have standardized txt files, opt to split each file separately
func parameterHeuristics(c echo.Context, sub *db.Submission, textFile *db.File, b *cache.Item, mu *sync.Mutex) error {
	var params utils.Params
	var err error
	f := &textFile.File
	c.Logger().Debugf("processing params for %s", f.FileName)

	baseConfig := utils.Config{Text: string(b.Blob), Filename: f.FileName}
	switch sub.UserID {
	case utils.IDAutoSnep:
		params, err = utils.AutoSnep(utils.WithConfig(baseConfig))
	case utils.IDDruge:
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseDruge(), utils.WithFilename(f.FileName))
	case utils.IDAIBean:
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseAIBean(), utils.WithFilename(f.FileName))
	case utils.IDArtieDragon:
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseArtie(), utils.WithFilename(f.FileName))
	case 1125540:
		hasFileName := func(line string) bool { return strings.HasPrefix(line, "File Name") }
		params, err = utils.Common(utils.WithConfig(baseConfig), utils.WithKeyCondition(hasFileName))
	case utils.IDFairyGarden:
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseFairyGarden(), utils.WithFilename(f.FileName))
	case utils.IDCirn0:
		params, err = utils.Cirn0(utils.WithConfig(baseConfig))
	case utils.IDHornybunny:
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseHornybunny(), utils.WithFilename(f.FileName))
	case utils.IDMethuzalach:
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseMethuzalach(), utils.WithFilename(f.FileName))
	case utils.IDSoph:
		if utils.SophStartInvokeAI.Match(b.Blob) {
			if param, err := utils.Soph(utils.WithConfig(baseConfig)); err == nil {
				mu.Lock()
				insertOrInitalize(&sub.Metadata.Objects, param)
				mu.Unlock()
				return nil
			}
		}
		params, err = utils.Common(utils.WithConfig(baseConfig), utils.UseSoph())
	case utils.IDNastAI:
		params, err = utils.Sequential(utils.WithConfig(baseConfig))
	default:
		params, err = utils.Common(
			utils.WithBytes(bytes.Join([][]byte{[]byte(f.FileName), b.Blob}, []byte("\n"))),
			utils.WithKeyCondition(func(line string) bool { return strings.HasPrefix(line, f.FileName) }))
	}
	if err != nil {
		return err
	}
	if len(params) > 0 {
		c.Logger().Debugf("finished params for %s", f.FileName)
		mu.Lock()
		insertOrInitalize(&sub.Metadata.Params, params)
		paramsToObject(c, sub, textFile)
		mu.Unlock()
	}
	return nil
}

func paramsToObject(c echo.Context, sub *db.Submission, textFile *db.File) {
	if sub.Metadata.Objects != nil {
		return
	}
	if sub.Metadata.Params == nil {
		return
	}

	var (
		wg           sync.WaitGroup
		mutex        sync.Mutex
		foundObjects bool
	)
	for fileName, params := range sub.Metadata.Params {
		for k, v := range params {
			if !strings.HasPrefix(k, utils.Objects) {
				continue
			}
			wg.Add(1)
			go func(name, object, content string) {
				defer wg.Done()
				item := &cache.Item{
					Blob:     []byte(content),
					MimeType: echo.MIMEApplicationJSON,
				}
				textFile := *textFile
				textFile.File.FileName = fmt.Sprintf("%s (%s)", name, object)
				if !jsonHeuristics(c, sub, item, &textFile, &mutex) {
					return
				}
				mutex.Lock()
				delete(params, object)
				mutex.Unlock()
				foundObjects = true
			}(fileName, k, v)
		}
	}
	wg.Wait()
	if foundObjects {
		return
	}
	for fileName, params := range sub.Metadata.Params {
		if p, ok := params[utils.Parameters]; ok {
			c.Logger().Debugf("processing heuristics for %s", fileName)
			wg.Add(1)
			go func(name string, content string) {
				defer wg.Done()
				heuristics, err := utils.ParameterHeuristics(content)
				if err != nil {
					c.Logger().Errorf("error processing heuristics for %s: %v", name, err)
					return
				}
				if tool := PrivateTools.FindString(p); tool != "" {
					sub.Metadata.AISubmission = true
					sub.Metadata.PrivateTool = true
					sub.Metadata.Generator = tool
				}
				mutex.Lock()
				insertOrInitalize(&sub.Metadata.Objects, map[string]entities.TextToImageRequest{name: heuristics})
				mutex.Unlock()
			}(fileName, p)
		}
	}
	wg.Wait()
}

func processDescriptionHeuristics(c echo.Context, sub *db.Submission) {
	c.Logger().Debugf("processing description heuristics for %s", sub.URL)
	var heuristics entities.TextToImageRequest
	var err error
	switch sub.UserID {
	case utils.IDRNSDAI:
		heuristics, err = utils.RNSDAIHeuristics(sub.Description)
	default:
		heuristics, err = utils.DescriptionHeuristics(sub.Description)
	}
	if err != nil {
		c.Logger().Errorf("error processing description heuristics for %s: %v", sub.URL, err)
		return
	}
	if reflect.DeepEqual(heuristics, entities.TextToImageRequest{}) {
		c.Logger().Debugf("no heuristics found for %s", sub.URL)
		return
	}
	insertOrInitalize(&sub.Metadata.Objects, map[string]entities.TextToImageRequest{sub.Title: heuristics})
}

package service

import (
	"encoding/json"
	"fmt"
	"github.com/ellypaws/inkbunny-app/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/ellypaws/inkbunny-sd/entities"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
	"github.com/labstack/echo/v4"
	"path/filepath"
	"strings"
	"time"
)

func QueryHost(c echo.Context, cacheToUse cache.Cache, host *sd.Host, database *db.Sqlite, hash string) (db.ModelHashes, error) {
	var knownModels []entities.Lora
	knownLorasKey := fmt.Sprintf("%v:loras", echo.MIMEApplicationJSON)
	item, err := cacheToUse.Get(knownLorasKey)
	if err == nil {
		c.Logger().Debugf("Cache hit for %s", knownLorasKey)
		if err := json.Unmarshal(item.Blob, &knownModels); err != nil {
			return nil, err
		}
	} else {
		c.Logger().Warnf("Cache miss for %s, retrieving known models...", knownLorasKey)

		knownModels, err = host.GetLoras()
		if err != nil {
			return nil, err
		}

		bin, err := json.Marshal(knownModels)
		if err != nil {
			return nil, err
		}

		err = cacheToUse.Set(knownLorasKey, &cache.Item{
			Blob:       bin,
			LastAccess: time.Now().UTC(),
			MimeType:   echo.MIMEApplicationJSON,
		}, cache.Day)
		if err != nil {
			c.Logger().Errorf("error caching known models: %v", err)
			return nil, err
		}
	}

	recache := c.QueryParam("recache") == "true"

	var match db.ModelHashes
	for _, lora := range knownModels {
		if h := lora.Metadata.SshsModelHash; h != nil && *h == "" {
			c.Logger().Warnf("model %s contained an empty hash, calculating...", lora.Name)
			lora.Metadata.SshsModelHash = nil
		}

		autoV3, err := RetrieveHash(c, cacheToUse, lora)
		if err != nil {
			continue
		}

		if autoV3 == "" {
			c.Logger().Warnf("model %s does not have a hash, skipping...", lora.Name)
			continue
		}

		if !recache {
			if hash != autoV3 {
				continue
			}
		}

		names := []string{lora.Name}
		if lora.Alias != lora.Name &&
			lora.Alias != "None" &&
			lora.Alias != "" {
			names = append(names, lora.Alias)
		}
		if lora.Path != "" {
			names = append(names, filepath.Base(strings.ReplaceAll(lora.Path, "\\", "/")))
		}
		h := db.ModelHashes{autoV3: names}

		if hash == autoV3 {
			match = h
			if !recache {
				break
			}
		}

		if err := database.UpsertModel(db.ModelHashes{autoV3: names}); err != nil {
			return nil, err
		}
	}

	return match, nil
}

func RetrieveHash(c echo.Context, cacheToUse cache.Cache, lora entities.Lora) (string, error) {
	if h := lora.Metadata.SshsModelHash; h != nil && *h != "" {
		if len(*h) > 12 {
			return (*h)[:12], nil
		}
		return *h, nil
	}

	key := fmt.Sprintf("%v:hash:%v", echo.MIMETextPlain, filepath.Base(strings.ReplaceAll(lora.Path, "\\", "/")))

	item, err := cacheToUse.Get(key)
	if err == nil {
		c.Logger().Debugf("Cache hit for %s", key)
		if len(item.Blob) > 12 {
			return string(item.Blob)[:12], nil
		}
		return string(item.Blob), nil
	} else {
		c.Logger().Warnf("Cache miss for %s, calculating hash...", key)
		hash, err := sd.LoraSafetensorHash(lora.Path)
		if err != nil {
			c.Logger().Errorf("error calculating hash for %s: %v", lora.Name, err)
			return "", err
		}

		err = cacheToUse.Set(key, &cache.Item{
			Blob:       []byte(hash.AutoV3Full),
			LastAccess: time.Now().UTC(),
			MimeType:   echo.MIMETextPlain,
		}, cache.Indefinite)
		if err != nil {
			c.Logger().Errorf("error caching hash %s: %v", key, err)
		} else {
			c.Logger().Infof("Cached %s %s %dB", key, echo.MIMETextPlain, len(hash.AutoV3Full))
		}

		return hash.AutoV3, nil
	}
}

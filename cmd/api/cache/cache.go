package cache

import (
	"bytes"
	"encoding/json"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/labstack/echo/v4"
	units "github.com/labstack/gommon/bytes"
	"math"
	"net/http"
	"time"
)

type Cache interface {
	Get(c echo.Context, key string) (*Item, error)
	Set(c echo.Context, key string, item *Item) error
}

type Item struct {
	Blob       []byte    `json:"blob,omitempty"`
	LastAccess time.Time `json:"last_access"`
	MimeType   string    `json:"mime_type,omitempty"`
	HitCount   int       `json:"hit_count,omitempty"`
}

func (item *Item) MarshalBinary() ([]byte, error) {
	if item.MimeType == echo.MIMEApplicationJSON {
		blob := item.Blob
		item.Blob = nil
		b, err := json.Marshal(item)
		if err != nil {
			return b, err
		}
		return bytes.Join(
			[][]byte{
				b[:len(b)-1],
				[]byte(`,"blob":`),
				blob,
				[]byte("}"),
			}, nil), nil
	}
	return json.Marshal(item)
}

func (item *Item) UnmarshalBinary(b []byte) error {
	return json.Unmarshal(b, item)
}

func (item *Item) Accessed() {
	backoff := int64(min(math.Pow(2, float64(item.HitCount-1)), 24*time.Hour.Seconds()))
	item.LastAccess = time.Now().Add(time.Duration(backoff) * time.Second)
	item.HitCount += 1
}

func Retrieve(c echo.Context, cache Cache, key string) (*Item, func(c echo.Context) error) {
	item, err := cache.Get(c, key)
	if err != nil {
		return nil, errFunc(http.StatusInternalServerError, err)
	}

	c.Logger().Debugf("retrieved %s %s %dKiB", key, item.MimeType, len(item.Blob)/units.KiB)
	c.Response().Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
	return item, nil
}

func errFunc(r int, err error) func(c echo.Context) error {
	return func(c echo.Context) error {
		return c.JSON(r, crashy.Wrap(err))
	}
}

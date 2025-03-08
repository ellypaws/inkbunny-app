package civitai

import (
	"net/url"

	"github.com/ellypaws/inkbunny-app/pkg/api/library"
)

var DefaultHost = &Host{
	Scheme: "https",
	Host:   "civitai.com",
}

type Host url.URL

// GetByHash https://civitai.com/api/v1/model-versions/by-hash/:hash
// [Documentation]
//
// [Documentation]: https://github.com/civitai/civitai/wiki/REST-API-Reference#get-apiv1models-versionsmodelversionid
func (h *Host) GetByHash(hash string) (*CivitAIModel, error) {
	u := As(h).WithPath("/api/v1/model-versions/by-hash/" + hash)
	var model CivitAIModel
	err := library.Get(u, library.WithDest(&model))
	return &model, err
}

// GetByModelID https://civitai.com/api/v1/model-versions/:id
// [Documentation]
//
// [Documentation]: https://github.com/civitai/civitai/wiki/REST-API-Reference#get-apiv1models-versionsmodelversionid
func (h *Host) GetByModelID(id string) (*CivitAIModel, error) {
	u := As(h).WithPath("/api/v1/model-versions/" + id)
	var model CivitAIModel
	err := library.Get(u, library.WithDest(&model))
	return &model, err
}

func As(h *Host) *library.Host {
	return (*library.Host)(h)
}

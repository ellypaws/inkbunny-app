package entities

import "encoding/json"

type Loras []Lora

func UnmarshalLoras(data []byte) ([]Lora, error) {
	var r Loras
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Loras) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Lora struct {
	Name     string   `json:"name"`
	Alias    string   `json:"alias"`
	Path     string   `json:"path"`
	Metadata Metadata `json:"metadata"`
}

type Metadata struct {
	SsSDModelName    *string           `json:"ss_sd_model_name,omitempty"`
	SsClipSkip       *string           `json:"ss_clip_skip,omitempty"`
	SsMixedPrecision *SsMixedPrecision `json:"ss_mixed_precision,omitempty"`

	// SsNewSDModelHash SHA256 / Full AutoV2 is usually the checkpoint hash used to train the lora
	SsNewSDModelHash *string `json:"ss_new_sd_model_hash,omitempty"`
	// Deprecated: AutoV1, use SsNewSDModelHash.
	// This is the old checkpoint hash.
	SsSDModelHash  *string `json:"ss_sd_model_hash,omitempty"`
	SshsLegacyHash *string `json:"sshs_legacy_hash,omitempty"`

	// SshsModelHash AutoV3 is usually the correct hash to use.
	// This is the whole SHA256 with the headers skipped.
	// Only use the first 12 bytes for short hash.
	SshsModelHash *string `json:"sshs_model_hash,omitempty"`

	SsNewVaeHash *SsNewVaeHash `json:"ss_new_vae_hash,omitempty"`
	SsVaeHash    *SsVaeHash    `json:"ss_vae_hash,omitempty"`
	SsVaeName    *string       `json:"ss_vae_name,omitempty"`
}

type SsMixedPrecision string

type SsNewVaeHash string

const (
	VaeHashC6A580B13A5B SsNewVaeHash = "c6a580b13a5bc05a5e16e4dbb80608ff2ec251a162311590c1f34c013d7f3dab"
	VaeHashF921FB3F2989 SsNewVaeHash = "f921fb3f29891d2a77a6571e56b8b5052420d2884129517a333c60b1b4816cdf"
	VaeHash2F11C4A99DDC SsNewVaeHash = "2f11c4a99ddc28d0ad8bce0acc38bed310b45d38a3fe4bb367dc30f3ef1a4868"
	VaeHash63AEECb90FF7 SsNewVaeHash = "63aeecb90ff7bc1c115395962d3e803571385b61938377bc7089b36e81e92e2e"
)

type SsVaeHash string

const (
	D636E597    SsVaeHash = "d636e597"
	F458B5C6    SsVaeHash = "f458b5c6"
	The223531C6 SsVaeHash = "223531c6"
	The975B2546 SsVaeHash = "975b2546"
)

// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    civitAIModel, err := UnmarshalCivitAIModel(bytes)
//    bytes, err = civitAIModel.Marshal()

package civitai

import "encoding/json"

func UnmarshalCivitAIModel(data []byte) (CivitAIModel, error) {
	var r CivitAIModel
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *CivitAIModel) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type CivitAIModel struct {
	ID                   int64    `json:"id"`
	ModelID              int64    `json:"modelId"`
	Name                 string   `json:"name"`
	CreatedAt            string   `json:"createdAt"`
	UpdatedAt            string   `json:"updatedAt"`
	Status               string   `json:"status"`
	PublishedAt          string   `json:"publishedAt"`
	TrainedWords         []string `json:"trainedWords"`
	TrainingStatus       any      `json:"trainingStatus"`
	TrainingDetails      any      `json:"trainingDetails"`
	BaseModel            string   `json:"baseModel"`
	BaseModelType        string   `json:"baseModelType"`
	EarlyAccessTimeFrame int64    `json:"earlyAccessTimeFrame"`
	Description          *string  `json:"description,omitempty"`
	Stats                Stats    `json:"stats"`
	Model                Model    `json:"model"`
	Files                []File   `json:"files"`
	Images               []Image  `json:"images"`
	DownloadURL          string   `json:"downloadUrl"`
}

type File struct {
	ID                int64        `json:"id"`
	SizeKB            float64      `json:"sizeKB"`
	Name              string       `json:"name"`
	Type              string       `json:"type"`
	PickleScanResult  string       `json:"pickleScanResult"`
	PickleScanMessage string       `json:"pickleScanMessage"`
	VirusScanResult   string       `json:"virusScanResult"`
	VirusScanMessage  *string      `json:"virusScanMessage,omitempty"`
	ScannedAt         string       `json:"scannedAt"`
	Metadata          FileMetadata `json:"metadata"`
	Hashes            Hashes       `json:"hashes"`
	Primary           bool         `json:"primary"`
	DownloadURL       string       `json:"downloadUrl"`
}

type Hashes struct {
	// Deprecated: Old AutoV1 hash where it only hashes a part of the file.
	AutoV1 string `json:"AutoV1"`

	// AutoV2 is the short hash usually used for checkpoints.
	// Only use the first 10 bytes for short hash.
	AutoV2 string `json:"AutoV2"`
	// Full SHA256 hash of the full file without skipping headers.
	Sha256 string `json:"SHA256"`

	Crc32  string `json:"CRC32"`
	Blake3 string `json:"BLAKE3"`

	// AutoV3 is usually the correct hash to use for loras.
	// This is SHA256 with the headers skipped.
	// Only use the first 12 bytes for short hash.
	AutoV3 string `json:"AutoV3"`
}

type FileMetadata struct {
	Format *FormatEnum `json:"format,omitempty"`
	Size   *SizeEnum   `json:"size,omitempty"`
	FP     *FPEnum     `json:"fp,omitempty"`
}

type FormatEnum = string
type SizeEnum = string
type FPEnum = string

const (
	FormatSafeTensor   FormatEnum = "SafeTensor"
	FormatPickleTensor FormatEnum = "PickleTensor"
	FormatOther        FormatEnum = "Other"
)

const (
	SizeFull   SizeEnum = "full"
	SizePruned SizeEnum = "pruned"
)

const (
	FP16 FPEnum = "fp16"
	FP32 FPEnum = "fp32"
)

type Image struct {
	URL          string        `json:"url"`
	NsfwLevel    int64         `json:"nsfwLevel"`
	Width        int64         `json:"width"`
	Height       int64         `json:"height"`
	Hash         string        `json:"hash"`
	Type         string        `json:"type"`
	Metadata     ImageMetadata `json:"metadata"`
	Availability string        `json:"availability"`
	Meta         any           `json:"meta,omitempty"`
}

type ImageMetadata struct {
	Hash   string `json:"hash"`
	Width  int64  `json:"width"`
	Height int64  `json:"height"`
}

type Model struct {
	Name string    `json:"name"`
	Type TypeEnum  `json:"type"`
	Nsfw bool      `json:"nsfw"`
	Poi  bool      `json:"poi"`
	Mode *ModeEnum `json:"mode,omitempty"`
}

type TypeEnum = string
type ModeEnum = string

const (
	TypeCheckpoint        TypeEnum = "Checkpoint"
	TypeTextualInversion  TypeEnum = "TextualInversion"
	TypeHypernetwork      TypeEnum = "Hypernetwork"
	TypeAestheticGradient TypeEnum = "AestheticGradient"
	TypeLORA              TypeEnum = "LORA"
	TypeControlnet        TypeEnum = "Controlnet"
	TypePoses             TypeEnum = "Poses"
)

const (
	ModeArchived  ModeEnum = "Archived"
	ModeTakenDown ModeEnum = "TakenDown"
)

type Stats struct {
	DownloadCount int64   `json:"downloadCount"`
	RatingCount   int64   `json:"ratingCount"`
	Rating        float64 `json:"rating"`
	ThumbsUpCount int64   `json:"thumbsUpCount"`
}

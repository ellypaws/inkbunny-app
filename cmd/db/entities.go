package db

import (
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny/api"
	"time"
)

// Flag is the type of violation that had incurred in a Submission
// In the event that the image is mostly or fully AI-generated:
//   - The image must be tagged with the ai_generated keyword
//   - The image must be tagged with the name of the generator and model used
//   - The image description must contain all the prompts and seeds passed to the generator
//   - The image description must indicate what generator and model was used
//   - The image description must indicate what training data was used (if known)
//   - The image must not have been generated using prompts that include the name of a living or recently deceased (within the last 25 years) artist, or the names of specific non-commercial characters (fursonas) without the express permission of the character owner - in line with our Ownership policy
//   - You must not sell fully AI-generated artwork adopts or commissions
//   - You may not upload more than six images with the same prompts in a single set
//     (this means "be selective", not "make lots of submissions with the same prompts")
//
// In the event that you used an AI tool to assist in the creation of assets or backgrounds for an otherwise manually-created piece of artwork:
//   - The image must be tagged with the ai_assisted keyword
//   - You must indicate what parts of the image were AI generated in the description and follow the above rules in relation to keywords and descriptions
//   - You may sell artwork with AI-generated backgrounds or assets, however it must be made clear that the image contains (or will contain) AI generated components
type Flag string

const (
	FlagMissingPrompt Flag = "missing_prompt"
	FlagAUPViolation  Flag = "aup_violation"
	FlagMissingTags   Flag = "missing_tags"
	FlagMissingSeed   Flag = "missing_seed"
	FlagMissingModel  Flag = "missing_model"
	FlagArtistUsed    Flag = "artist_used"
	FlagCharacterUsed Flag = "character_used"
	FlagPrivateModel  Flag = "private_model"
	FlagPrivateLora   Flag = "private_lora"
	FlagPrivateTool   Flag = "private_tool"
	FlagSoldArt       Flag = "sold_art"
	FlagUndisclosed   Flag = "undisclosed"
	FlagTooMany       Flag = "too_many"

	// FlagMismatched is a Flag when the prompt do not generate close to the Submission
	FlagMismatched Flag = "mismatched"
)

type Audit struct {
	Auditor            *Auditor `json:"auditor"`
	SubmissionID       string   `json:"submission"`
	SubmissionUsername string   `json:"submission_username"` // The username of the user who submitted the image
	SubmissionUserID   string   `json:"submission_user_id"`  // The user ID of the user who submitted the image
	Flags              []Flag   `json:"flags"`
	ActionTaken        string   `json:"action_taken"`
}

type Auditor struct {
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	Role       Role   `json:"role"`
	AuditCount int    `json:"audit_count"`
}

type Role string

const (
	RoleAdmin        Role = "administrator"
	RoleAuditor      Role = "auditor"
	RoleCommunityMod Role = "community_mod"
	RoleSuperMod     Role = "super_mod"
	RoleModerator    Role = "moderator" // Deprecated: use SuperMod
	RoleUser         Role = "user"
)

const (
	StableDiffusion = "stable_diffusion"
)

type GenerationInfo struct {
	Generator   string                        `json:"generator"`
	Model       string                        `json:"model"`
	TextToImage *entities.TextToImageRequest  `json:"text_to_image,omitempty"`
	Img2Img     *entities.ImageToImageRequest `json:"image_to_image,omitempty"`
}

type Submission struct {
	ID          string               `json:"id"`
	UserID      string               `json:"user_id"`
	URL         string               `json:"url"`
	Audit       *Audit               `json:"audit,omitempty"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Updated     time.Time            `json:"updated_at"`
	Generated   bool                 `json:"generated"`
	Assisted    bool                 `json:"assisted"`
	Img2Img     bool                 `json:"img2img"`
	Ratings     api.SubmissionRating `json:"ratings"`
	Keywords    []Keyword            `json:"keywords,omitempty"`
	Files       []File               `json:"files,omitempty"`
}

type File struct {
	File api.File        `json:"file"`
	Info *GenerationInfo `json:"info,omitempty"`
	Blob *string         `json:"blob,omitempty"`
}

type Keyword struct {
	KeywordID   string `json:"keyword_id"`
	KeywordName string `json:"keyword_name"`
	Suggested   bool   `json:"contributed"`
}

package db

import (
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny/api"
	"time"
)

type Flag string

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
//
// From https://wiki.inkbunny.net/wiki/ACP#AI
//
// # Use of open-source AI tools combined with freely-available models is permitted, with appropriate keywords and limits on excessive or commercial use
//
// Inkbunny wants to help its members share and enjoy AI experiments and knowledge and benefit from assistance with tedious tasks, while limiting the impact on existing creators and the site in general, and discouraging proprietary tools or services based on harvesting publicly-accessible work to create a walled garden.
//
// For all AI-generated or AI-assisted content:
//   - You must not post work using closed-source tools or services that do not make their code and models freely available for others to reuse in an equivalent manner
//   - This includes services such as Midjourney and NovelAI that are based on proprietary models
//   - You must not use the names of living or recently-deceased creators (within the last 25 years) or their non-commercial characters as prompts without their permission, nor train models and/or use artist-focused LoRAs to obtain a similar effect
//   - The description must contain all prompts, seeds and LoRAs passed to AI tools and must indicate the generator, training model and version or hash used - for advanced projects that cannot be described in this manner, attach the workflow JSON as an additional file
//
// If your work is mostly or fully generated by AI:
//   - The work must be tagged with the ai_generated keyword, the name of the tool and model that was used
//   - You must not post submissions offering such content for commission or paid adoption
//   - You must use a multi-file submission containing no more than six pieces of work for work generated via the same prompt
//
// If you used an AI tool to assist in the creation of assets or backgrounds for an otherwise manually-created work:
//   - The image must be tagged with the ai_assisted keyword, the name of the tool and model that was used
//   - You must indicate what parts of the work were AI generated in the description
//   - You may sell and offer commissions for content using AI-generated assets or backgrounds, but you must notify customers of its use beforehand
//
// If you used an AI tool to produce assisted output from input you created (eg. img2img):
//   - The image must be tagged with the ai_assisted keyword
//   - You must include the original input as part of the submission
//   - We don't require every subsequent hand-drawn input, but a viewer should understand how the end result was obtained through use of the tools
//   - If initially created in a recorded stream, a link may be helpful
//
// If you used an AI tool to modify your own work, such as frame interpolation or upscaling:
//   - You must include the original input as part of the submission or as a scrap
//   - No extra sale restrictions or keyword requirements apply
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
	id                 int64
	auditor            Auditor
	AuditorID          *int64 `json:"auditor_id"`
	SubmissionID       int64  `json:"submission"`
	SubmissionUsername string `json:"submission_username"` // The username of the user who submitted the image
	SubmissionUserID   int64  `json:"submission_user_id"`  // The user ID of the user who submitted the image
	Flags              []Flag `json:"flags"`
	ActionTaken        string `json:"action_taken"`
}

func (a *Audit) ID() int64 {
	return a.id
}

func (a *Audit) Auditor() Auditor {
	return a.auditor
}

type Auditor struct {
	UserID     int64  `json:"user_id"`
	Username   string `json:"username"`
	Role       Role   `json:"role"`
	AuditCount int    `json:"audit_count"`
}

type Role int

// Role is the type of role that an Auditor can have
// Admin is the highest role, and has the ability to perform all actions
// The list of Inkbunny staff can be found at https://inkbunny.net/adminsmods_process.php
const (
	RoleAdmin Role = iota
	RoleCommunityMod
	RoleSuperMod
	RoleModerator // Deprecated: use SuperMod
	RoleAuditor
	RoleUser
)

func (r Role) String() string {
	switch r {
	case RoleAdmin:
		return "admin"
	case RoleCommunityMod:
		return "community_mod"
	case RoleSuperMod:
		return "super_mod"
	case RoleModerator:
		return "moderator"
	case RoleAuditor:
		return "auditor"
	case RoleUser:
		return "user"
	default:
		return "unknown"
	}
}

func RoleLevel(s string) Role {
	switch s {
	case "admin":
		return RoleAdmin
	case "community_mod":
		return RoleCommunityMod
	case "super_mod":
		return RoleSuperMod
	case "moderator":
		return RoleModerator
	case "auditor":
		return RoleAuditor
	case "user":
		return RoleUser
	default:
		return RoleUser
	}
}

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
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	URL         string `json:"url"`
	audit       *Audit
	AuditID     *int64                 `json:"audit_id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Updated     time.Time              `json:"updated_at"`
	Generated   bool                   `json:"generated"`
	Assisted    bool                   `json:"assisted"`
	Img2Img     bool                   `json:"img2img"`
	Ratings     []api.SubmissionRating `json:"ratings"`
	Keywords    []api.Keyword          `json:"keywords,omitempty"`
	Files       []File                 `json:"files,omitempty"`
}

func (s *Submission) Audit() *Audit {
	return s.audit
}

type File struct {
	File api.File        `json:"file"`
	Info *GenerationInfo `json:"info,omitempty"`
	Blob *string         `json:"blob,omitempty"`
}

type SIDHash struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	hashes   hashmap
}

type hashmap map[string]struct{}

type ModelHashes map[string][]string

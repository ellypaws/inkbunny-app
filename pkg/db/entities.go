package db

import (
	"time"

	"github.com/ellypaws/inkbunny/api"

	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/utils"
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

// These describe the general acceptable content policy of Inkbunny from the [official wiki]
// Inkbunny is an art community for sharing your best work.
// We welcome sketches, works in progress, and other incomplete work,
// but we encourage you to dedicate most of your gallery to your best completed creations.
//
// This Acceptable Content Policy forms part of the overall [Terms of Service].
//
// [official wiki]: https://wiki.inkbunny.net/wiki/ACP
// [Terms of Service]: https://inkbunny.net/tos.php
const (
	// FlagOwnership is a [Flag] for a [Ticket] that does not have the proper ownership of the artwork.
	//
	// The work you upload must be created by you, or for you.
	// If you did not create the artwork and it was created for you then you must indicate in the Description who created it.
	// You must be the copyright owner of the artwork and all the characters they contain, or you must have permission from the copyright owners to post their art or characters.
	// https://wiki.inkbunny.net/wiki/ACP#Ownership
	FlagOwnership Flag = "ownership"
	// FlagCommercial is a [Flag] for a [Ticket] that does not have proper commercial rights of the artwork.
	// "Fan art" of commercial copyright characters is allowed in free uploads,
	// provided you do not sell the work on Inkbunny and you indicate who owns the characters.
	// https://wiki.inkbunny.net/wiki/ACP#Ownership
	FlagCommercial Flag = "fan_art"
	// FlagDerivative is a [Flag] for a [Ticket] that does not have enough changes to be considered a new creation.
	//
	// Posting submissions that contain portions of other artists'
	// work (such as using them for backgrounds or other components)
	// is allowed only if you have received their permission to do so.
	// The works you create using portions of other artists'
	// work must be sufficiently unique to be considered a new creation.
	//
	// Posting re-colors or shading of other artists' work is allowed if they have given permission directly to you,
	// and when it is clear you put in significant effort to change or enhance that work.
	// Simply adjusting hue and color balance values, or other superficial changes are not sufficient.
	// https://wiki.inkbunny.net/wiki/ACP#Derivative_Works
	FlagDerivative Flag = "permission"
	// FlagTracing is a [Flag] for a [Ticket] where the artwork is traced from another artist's work.
	//
	// Using portions of other artists' work,
	// such as [Tracing], is not permitted without the artist's permission.
	//
	// [Tracing]: https://wiki.inkbunny.net/wiki/ACP#Tracing
	FlagTracing Flag = "tracing"
	// FlagSampling is a [Flag] for a [Ticket] that is not significantly original.
	//
	// Sampling tracks for use in music or creating "mashups"
	// is permitted in audio submissions as long as the work you create is something significantly original,
	// and you credit the source.
	// Where possible, you should seek permission from the source.
	// https://wiki.inkbunny.net/wiki/ACP#Remixes_and_Mashups
	FlagSampling Flag = "ai_sampling"
	// FlagHumanContent is a [Flag] for a [Ticket] that does not follow the guidelines for [human] content.
	//
	// Human characters are permitted in artwork,
	// however they must not appear in sexual situations and must not show genitals, anal details, or sexual arousal.
	// Censored art involving humans must plausibly depict a non-sexual situation.
	//
	// Human characters are permitted in stories only so long as they are not involved in sexual situations of any kind.
	// This policy also applies to thumbnails for stories and music.
	//
	// Characters that are essentially [human] (pixies, faeries, elves, orcs, trolls, etc)
	// or just have ears/tails or other superficial animal features applied are considered human for this rule.
	//
	// [human]: https://wiki.inkbunny.net/wiki/ACP#Human_Characters
	FlagHumanContent Flag = "human_content"
	FlagPhotography  Flag = "photography"
	// FlagAIGenerated is a [Flag] for a [Ticket] that does not follow the guidelines for [AI] generated content.
	//
	// Use of open-source [AI] tools combined with freely-available models is permitted,
	// with appropriate keywords and limits on excessive or commercial use
	// Verbose [Flag] are available above such as but not limited to
	// [FlagMissingPrompt], [FlagMissingTags], [FlagPrivateModel]
	//
	// [AI]: https://wiki.inkbunny.net/wiki/ACP#AI
	FlagAIGenerated    Flag = "ai_generated"
	FlagAIAssisted     Flag = "ai_assisted"
	FlagVideoContent   Flag = "video_content"
	FlagHarassment     Flag = "harassment"
	FlagGameScreenshot Flag = "game_screenshot"
	FlagMovieContent   Flag = "movie_content"
	FlagUserIcon       Flag = "user_icon"
	// FlagKeywordPolicy is a [Flag] for a [Ticket] that does not follow the [Keyword Policy].
	//
	// All users must abide by the [Keyword Policy] which forms part of this [Acceptable Content Policy].
	//
	// [Keyword Policy]: https://wiki.inkbunny.net/wiki/ACP#Keyword_Policy
	// [Acceptable Content Policy]: https://wiki.inkbunny.net/wiki/ACP
	FlagKeywordPolicy Flag = "keyword_policy"
	// FlagContentRepost is a [Flag] for a [Ticket] that reposts too much or too frequently.
	//
	// The same work must not be posted to your own account more than once within 72 hours,
	// or more than three times in total, regardless of whether other posts are subsequently deleted.
	// Journals referencing or [including thumbnails of submissions] do not count for this purpose.
	//
	// [including thumbnails of submissions]: https://wiki.inkbunny.net/wiki/BBCode#Linking_to_Images
	//
	// https://wiki.inkbunny.net/wiki/ACP#Reposting_and_Reminders
	FlagContentRepost Flag = "content_repost"
	FlagSiteHarm      Flag = "site_harm"
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
// The submissions of Inkbunny staff can be found at https://inkbunny.net/adminsmods_process.php
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

func (r Role) IsAuditor() bool {
	return r <= RoleAuditor
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
	Username    string `json:"username,omitempty"`
	URL         string `json:"url"`
	audit       *Audit
	AuditID     *int64                 `json:"audit_id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Updated     time.Time              `json:"updated_at"`
	Metadata    Metadata               `json:"metadata,omitempty"`
	Ratings     []api.SubmissionRating `json:"ratings"`
	Keywords    []api.Keyword          `json:"keywords,omitempty"`
	Files       []File                 `json:"files,omitempty"`
}

type Metadata struct {
	Generated bool `json:"generated"`
	Assisted  bool `json:"assisted"`
	Img2Img   bool `json:"img2img"` // includes inpaint

	HasJSON bool `json:"has_json"`
	HasTxt  bool `json:"has_txt"`

	StableDiffusion bool `json:"stable_diffusion"`
	ComfyUI         bool `json:"comfy_ui"`
	MultipleImages  bool `json:"multiple_images"`

	TaggedHuman     bool    `json:"tagged_human"`
	DetectedHuman   bool    `json:"detected_human"`
	HumanConfidence float64 `json:"human_confidence"`

	AITitle       bool     `json:"ai_title"`
	AIDescription bool     `json:"ai_description"`
	AIKeywords    []string `json:"ai_keywords,omitempty"`
	AIAccount     bool     `json:"ai_account"`
	AISubmission  bool     `json:"ai_submission"`

	// FlagMissingPrompt.
	// This gets set to false if at least one prompt is found.
	MissingPrompt bool `json:"missing_prompt"`
	// FlagMissingModel
	// This gets set to false if at least one model is found.
	MissingModel bool     `json:"missing_model"`
	MissingTags  bool     `json:"missing_tags"`           // FlagMissingTags
	ArtistUsed   []Artist `json:"artists_used,omitempty"` // FlagArtistUsed
	PrivateModel bool     `json:"private_model"`          // FlagPrivateModel
	PrivateLora  bool     `json:"private_lora"`           // FlagPrivateLora
	PrivateTool  bool     `json:"private_tool"`           // FlagPrivateTool
	SoldArt      bool     `json:"sold_art"`               // FlagSoldArt

	Generator string `json:"generator,omitempty"`

	Params utils.Params `json:"params,omitempty"`

	Objects map[string]entities.TextToImageRequest `json:"objects,omitempty"`
}

func (s *Submission) Audit() *Audit {
	return s.audit
}

type File struct {
	File    api.File              `json:"file"`
	Caption *entities.CaptionEnum `json:"caption,omitempty"`
}

type SIDHash struct {
	Hash      string `json:"sid_hash"`
	AuditorID int64  `json:"auditor_id"`
}

type HashID map[string]int64

type ModelHashes map[string][]string

type Ticket struct {
	ID            int64         `json:"id,omitempty"`
	Subject       string        `json:"subject"`
	DateOpened    time.Time     `json:"date_opened"`
	DateClosed    *time.Time    `json:"date_closed,omitempty"`
	Status        string        `json:"status,omitempty"`
	Labels        []TicketLabel `json:"labels,omitempty"`
	Priority      string        `json:"priority"`
	Closed        bool          `json:"closed"`
	Flags         []Flag        `json:"flags,omitempty"`
	Responses     []Response    `json:"responses,omitempty"`
	SubmissionIDs []int64       `json:"submission_ids,omitempty"`
	auditor       *Auditor
	AssignedID    *int64   `json:"assigned_id,omitempty"` // Auditor ID
	UsersInvolved Involved `json:"involved"`
}

func (t Ticket) Auditor() *Auditor {
	return t.auditor
}

type TicketLabel string

const (
	LabelAIGenerated   TicketLabel = "ai_generated"
	LabelAIAssisted    TicketLabel = "ai_assisted"
	LabelImg2Img       TicketLabel = "img2img"
	LabelTaggedHuman   TicketLabel = "tagged_human"
	LabelDetectedHuman TicketLabel = "detected_human"
	LabelJSON          TicketLabel = "json"
	LabelTxt           TicketLabel = "txt"
	LabelCannotParse   TicketLabel = "cannot_parse" // Cannot parse the JSON or TXT file
	LabelMissingParams TicketLabel = "missing_params"
	LabelMissingPrompt TicketLabel = "missing_prompt"
	LabelMissingTags   TicketLabel = "missing_tags"
	LabelMissingSeed   TicketLabel = "missing_seed"
	LabelMissingModel  TicketLabel = "missing_model"
	LabelArtistUsed    TicketLabel = "artist_used"
	LabelPrivateModel  TicketLabel = "private_model"
	LabelPrivateLora   TicketLabel = "private_lora"
	LabelPrivateTool   TicketLabel = "private_tool"
	LabelSoldArt       TicketLabel = "sold_art"
	LabelPayMention    TicketLabel = "payment_mention"

	// LabelBeforeRuleRevision is a [TicketLabel] for submissions before November 21, 2022.
	// An [announcement] was made on 11/20/2022 21:13 UTC which revised the rules for AI submissions.
	// "Best effort" for sketches/prompts on work posted before November 21, but keywords are required.
	// Continuous [revisions] might have been made in the [ACP] since the original draft, which should be monitored.
	//
	// [announcement]: https://inkbunny.net/j/467389
	// [revisions]: https://wiki.inkbunny.net/w/index.php?title=ACP&diff=cur&oldid=1082
	// [ACP]: https://wiki.inkbunny.net/wiki/ACP#AI
	LabelBeforeRuleRevision TicketLabel = "before_rule_revision"
)

var Nov21 = time.Date(2022, time.November, 21, 0, 0, 0, 0, time.UTC)

type Response struct {
	SupportTeam bool           `json:"support_team"`
	User        api.UsernameID `json:"user"`
	Date        time.Time      `json:"date"`
	Message     string         `json:"message"`
}

type Involved struct {
	Reporter    api.UsernameID   `json:"reporter"`
	ReportedIDs []api.UsernameID `json:"reported,omitempty"`
}

type Artist struct {
	Username string `json:"username" query:"username"`
	UserID   *int64 `json:"user_id,omitempty" query:"user_id"`
}

const TicketDateLayout = "2006-01-02"

type TicketReport struct {
	Username   string
	ReportDate time.Time
	Report     []byte
}

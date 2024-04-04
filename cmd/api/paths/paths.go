package paths

// GetPaths
// "/":
// "/inkbunny/description":
// "/inkbunny/submission":
// "/inkbunny/submission/:ids"
// "/inkbunny/search":
// "/image":
// "/tickets/audits":
const (
	Base                  = "/"
	InkbunnyDescription   = "/inkbunny/description"
	InkbunnySubmission    = "/inkbunny/submission"
	InkbunnySubmissionIDs = "/inkbunny/submission/:ids"
	InkbunnySearch        = "/inkbunny/search"
	Image                 = "/image"
	TicketsAudits         = "/tickets/audits"
)

// PostPaths
// "/login":
// "/logout":
// "/validate":
// "/llm":
// "/llm/json":
// "/prefill":
// "/interrogate":
// "/interrogate/upload":
// "/sd/:path":
const (
	Login             = "/login"
	Logout            = "/logout"
	Validate          = "/validate"
	LLM               = "/llm"
	LLMJSON           = "/llm/json"
	Prefill           = "/prefill"
	Interrogate       = "/interrogate"
	InterrogateUpload = "/interrogate/upload"
	SDPath            = "/sd/:path"
)

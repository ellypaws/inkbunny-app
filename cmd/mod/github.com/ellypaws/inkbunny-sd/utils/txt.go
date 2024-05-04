package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ellypaws/inkbunny-sd/entities"
	"io"
	"os"
	"regexp"
	"strings"
)

type Params map[string]PNGChunk
type PNGChunk map[string]string

type Config struct {
	Text          string
	KeyCondition  func(string) bool
	SkipCondition func(string) bool
	Filename      string
}

type Processor func(...func(*Config)) (Params, error)

const (
	Parameters     = "parameters"
	Postprocessing = "postprocessing"
	Extras         = "extras"
	Caption        = "caption"

	IDAutoSnep    = 1004248
	IDDruge       = 151203
	IDArtieDragon = 1190392
	IDAIBean      = 147301
	IDFairyGarden = 215070
	IDCirn0       = 177167
	IDHornybunny  = 12499
	IDNeoncortex  = 14603
	IDMethuzalach = 1089071
	IDRNSDAI      = 1188211
)

// AutoSnep is a Processor that parses yaml like raw txt where each two spaces is a new dict
// It's mostly seen in multi-chunk parameter output from AutoSnep
func AutoSnep(opts ...func(*Config)) (Params, error) {
	var c Config
	for _, f := range opts {
		f(&c)
	}
	var chunks Params = make(Params)
	scanner := bufio.NewScanner(strings.NewReader(c.Text))

	const Software = "Software"

	var png string
	var key string
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		line := scanner.Text()

		indentLevel := len(line) - len(strings.TrimLeft(line, " "))

		switch indentLevel {
		case 0:
			if strings.HasSuffix(line, ":") {
				png = "AutoSnep_" + strings.TrimSuffix(line, ":")
				chunks[png] = make(PNGChunk)
			}
		case 2: // PNG text chunks:
			if strings.TrimSpace(line) == "PNG text chunks:" {
				chunks[png] = make(PNGChunk)
			}
		case 4: // parameters:
			key = strings.TrimSpace(strings.TrimSuffix(line, ":"))
		case 6:
			if len(chunks[png][key]) > 0 {
				chunks[png][key] += "\n"
			}
			chunks[png][key] += line[6:]
		default:

		}
	}

	if len(chunks) == 0 {
		return nil, errors.New("no chunks found")
	}

	return chunks, nil
}

var seedLine = regexp.MustCompile(`seed: (\d+)`)

func Cirn0(opts ...func(*Config)) (Params, error) {
	var c Config
	for _, f := range opts {
		f(&c)
	}
	var chunks Params = make(Params)
	scanner := bufio.NewScanner(strings.NewReader(c.Text))

	var steps, sampler, cfg, model string
	var key string
	var lastKey string
	var foundSeed bool
	var foundNegative bool
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		line := scanner.Text()
		line = strings.TrimSpace(line)

		switch {
		case len(line) == 0:
			continue

		case strings.HasPrefix(line, "sampler:"):
			sampler = strings.TrimPrefix(line, "sampler: ")
		case strings.HasPrefix(line, "cfg:"):
			cfg = strings.TrimPrefix(line, "cfg: ")
		case strings.HasPrefix(line, "steps:"):
			steps = strings.TrimPrefix(line, "steps: ")
		case strings.HasPrefix(line, "model:"):
			model = strings.TrimPrefix(line, "model: ")

		case strings.HasPrefix(line, "==="):
			key = c.Filename + line
			chunks[key] = make(PNGChunk)

			lastKey = strings.Trim(line, "= ")

		case strings.HasPrefix(line, "---"):
			if len(lastKey) == 0 {
				lastKey = "unknown"
			}
			key = fmt.Sprintf("--- %s%s ---", lastKey, strings.Trim(line, "- "))
			chunks[key] = make(PNGChunk)

		case len(key) == 0:
			continue

		case strings.HasPrefix(line, "keyword prompt:"):
			continue

		case foundNegative:
			foundNegative = false
			chunks[key][Parameters] += fmt.Sprintf("\nNegative Prompt: %s", line)

		case strings.HasPrefix(line, "negative prompt:"):
			foundNegative = true
			continue

		case seedLine.MatchString(line):
			chunks[key][Parameters] += fmt.Sprintf(
				"\nSteps: %s, Sampler: %s, CFG scale: %s, Seed: %s, Model: %s",
				steps,
				sampler,
				cfg,
				seedLine.FindStringSubmatch(line)[1],
				model,
			)
			key = ""

		case foundSeed:
			foundSeed = false
			chunks[key][Parameters] += fmt.Sprintf(
				"\nSteps: %s, Sampler: %s, CFG scale: %s, Seed: %s, Model: %s",
				steps,
				sampler,
				cfg,
				line,
				model,
			)
			key = ""

		case strings.HasPrefix(line, "seed:"):
			foundSeed = true

		default:
			if len(chunks[key][Parameters]) > 0 {
				chunks[key][Parameters] += "\n"
			}
			chunks[key][Parameters] += line
		}
	}

	if len(chunks) == 0 {
		return nil, errors.New("no chunks found")
	}

	return chunks, nil
}

var drugeMatchDigit = regexp.MustCompile(`(?m)^\d+`)

func UseDruge() func(*Config) {
	return func(c *Config) {
		c.KeyCondition = func(line string) bool {
			return drugeMatchDigit.MatchString(line)
		}
		c.Filename = "druge_"
		if !drugeMatchDigit.MatchString(c.Text) {
			c.Text = "1\n" + c.Text
		}
	}
}

func UseArtie() func(*Config) {
	return func(c *Config) {
		c.KeyCondition = func(line string) bool {
			return strings.HasSuffix(line, "Image")
		}
		c.Filename = "artiedragon_"
	}
}

var aiBeanKey = regexp.MustCompile(`(?i)^(image )?\d+`)

func UseAIBean() func(*Config) {
	return func(c *Config) {
		c.KeyCondition = func(line string) bool {
			return aiBeanKey.MatchString(line)
		}
		c.Filename = "AIBean_"
		c.SkipCondition = func(line string) bool {
			return line == "parameters"
		}
		if aiBeanKey.MatchString(c.Text) {
			return
		}
		if strings.HasPrefix(c.Text, "parameters") {
			c.Text = strings.Replace(c.Text, "parameters", "1", 1)
			return
		}
		c.Text = "1\n" + c.Text
	}
}

func UseFairyGarden() func(*Config) {
	return func(c *Config) {
		c.KeyCondition = func(line string) bool {
			return strings.HasPrefix(line, "photo")
		}
		c.Filename = "fairygarden_"
		// prepend "photo 1" to the input in case it's missing
		c.Text = "photo 1\n" + c.Text
	}
}

func UseCirn0() func(*Config) {
	return func(c *Config) {
		c.KeyCondition = func(line string) bool {
			return strings.HasPrefix(line, "===")
		}
		c.Filename = "cirn0_"

		var part string
		lines := strings.Split(c.Text, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "=== #") {
				part = strings.TrimPrefix(line, "=== #")
			}
			if strings.HasPrefix(line, "---") {
				lines[i] = fmt.Sprintf("=== Part #%s", part)
			}
		}
		c.Text = strings.Join(lines, "\n")
	}
}

func UseHornybunny() func(*Config) {
	return func(c *Config) {
		c.Text = "(1)\n" + c.Text
		c.KeyCondition = func(line string) bool {
			return regexp.MustCompile(`^\(\d+\)$`).MatchString(line)
		}
		c.Filename = "Hornybunny_"
		//c.Text = strings.ReplaceAll(c.Text, "----", "")
		//c.Text = strings.ReplaceAll(c.Text, "Original generation details", "")
		//c.Text = strings.ReplaceAll(c.Text, "Upscaling details", "")
		c.Text = strings.ReplaceAll(c.Text, "Positive Prompt: ", "")
		c.Text = strings.ReplaceAll(c.Text, "Other details: ", "")
		c.SkipCondition = func(line string) bool {
			switch line {
			case "----":
				return true
			case "==========":
				return true
			case "Original generation details":
				return true
			case "Upscaling details":
				return true
			default:
				return false
			}
		}
	}
}

var (
	methuzalachModel    = regexp.MustCompile(`Model: [^\n]+`)
	methuzalachNegative = regexp.MustCompile(`Negative prompts:\s*`)
	methuzalachSeed     = regexp.MustCompile(`\s*Seed: \D*?[,\s]`)
	methuzalachSteps    = regexp.MustCompile(`.*(Steps: \d+[^\n]*)`)
)

func UseMethuzalach() func(*Config) {
	return func(c *Config) {
		c.KeyCondition = func(line string) bool {
			return strings.HasPrefix(line, "Image")
		}

		model := methuzalachModel.FindString(c.Text)
		c.Text = methuzalachNegative.ReplaceAllString(c.Text, "Negative Prompt: ")
		c.Text = methuzalachSeed.ReplaceAllString(c.Text, "")
		c.Text = methuzalachSteps.ReplaceAllString(c.Text, `$1 `+model)
	}
}

func WithString(s string) func(*Config) {
	return func(c *Config) {
		c.Text = s
	}
}

func WithBytes(b []byte) func(*Config) {
	return func(c *Config) {
		c.Text = string(b)
	}
}

func WithConfig(config Config) func(*Config) {
	return func(c *Config) {
		*c = config
	}
}

func WithFilename(filename string) func(*Config) {
	return func(c *Config) {
		c.Filename = filename
	}
}

func WithKeyCondition(f func(string) bool) func(*Config) {
	return func(c *Config) {
		c.KeyCondition = f
	}
}

func Common(opts ...func(*Config)) (Params, error) {
	var c Config
	for _, f := range opts {
		f(&c)
	}
	if c.KeyCondition == nil {
		return nil, errors.New("condition for key is not set")
	}
	var chunks Params = make(Params)
	scanner := bufio.NewScanner(strings.NewReader(c.Text))

	var key string
	var negativePrompt string
	var extra string
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		line := scanner.Text()

		if c.SkipCondition != nil && c.SkipCondition(line) {
			continue
		}
		if len(line) == 0 {
			continue
		}
		if c.KeyCondition(line) {
			key = c.Filename + line
			chunks[key] = make(PNGChunk)
			continue
		}
		if len(key) == 0 {
			continue
		}
		if len(negativePrompt) > 0 {
			if !negativeHasText.MatchString(negativePrompt) {
				chunks[key][Parameters] += line
				continue
			}
			chunks[key][Parameters] += "\n" + line
			if stepsStart.MatchString(line) {
				negativePrompt = ""
				key = ""
			}
			continue
		}
		if negativeStart.MatchString(line) {
			negativePrompt = line
			chunks[key][Parameters] += "\n" + line
			continue
		}
		if len(extra) > 0 {
			chunks[key][extra] += line
			extra = ""
			continue
		}
		switch line {
		case Postprocessing:
			extra = Postprocessing
			continue
		case Extras:
			extra = Extras
			continue
		}
		if len(chunks[key][Parameters]) > 0 {
			chunks[key][Parameters] += "\n"
		}
		chunks[key][Parameters] += line
	}

	if len(chunks) == 0 {
		return nil, errors.New("no chunks found")
	}

	return chunks, nil
}

func ParseParams(p Params) map[string]entities.TextToImageRequest {
	var request map[string]entities.TextToImageRequest
	for file, chunk := range p {
		if params, ok := chunk[Parameters]; ok {
			r, err := ParameterHeuristics(params)
			if err != nil {
				continue
			}
			if request == nil {
				request = make(map[string]entities.TextToImageRequest)
			}
			request[file] = r
		}
	}
	return request
}

type NameContent map[string][]byte

// ParseDataset takes in a map of text and json files and returns a map of the combined data
// It uses the commonInstruction as a base and appends the input and response to it following completeSample
func ParseDataset(text, json NameContent) map[string][]byte {
	var dataset = make(map[string][]byte)
	for name, input := range text {
		var out bytes.Buffer
		out.WriteString(commonInstruction)
		out.WriteString("### Input:\n")
		out.WriteString(`The file name is: "`)

		// Because some artists already have standardized txt files, opt to split each file separately
		autoSnep := strings.Contains(name, "_AutoSnep_")
		druge := strings.Contains(name, "_druge_")
		aiBean := strings.Contains(name, "_AIBean_")
		artieDragon := strings.Contains(name, "_artiedragon_")
		picker52578 := strings.Contains(name, "_picker52578_")
		fairyGarden := strings.Contains(name, "_fairygarden_")
		if autoSnep || druge || aiBean || artieDragon || picker52578 || fairyGarden {
			var inputResponse map[string]InputResponse
			switch {
			case autoSnep:
				inputResponse = MapParams(AutoSnep, WithBytes(input))
			case druge:
				inputResponse = MapParams(Common, WithBytes(input), UseDruge())
			case aiBean:
				inputResponse = MapParams(Common, WithBytes(input), UseAIBean())
			case artieDragon:
				inputResponse = MapParams(Common, WithBytes(input), UseArtie())
			case picker52578:
				inputResponse = MapParams(
					Common,
					WithBytes(input),
					WithFilename("picker52578_"),
					WithKeyCondition(func(line string) bool { return strings.HasPrefix(line, "File Name") }))
			case fairyGarden:
				inputResponse = MapParams(
					Common,
					// prepend "photo 1" to the input in case it's missing
					WithBytes(bytes.Join([][]byte{[]byte("photo 1"), input}, []byte("\n"))),
					UseFairyGarden())
			}
			if inputResponse != nil {
				out := out.Bytes()
				for name, s := range inputResponse {
					var multi bytes.Buffer
					multi.Write(out)
					multi.WriteString(name)
					multi.WriteString("\"\n\n")

					if s.Input == "" {
						continue
					}

					multi.WriteString(s.Input)

					multi.WriteString("\n\n")
					multi.WriteString("### Response:\n")

					multi.Write(s.Response)
					dataset[name] = multi.Bytes()
				}
				continue
			}
		}

		out.WriteString(name)
		out.WriteString("\"\n\n")

		out.Write(input)

		out.WriteString("\n\n")
		out.WriteString("### Response:\n")
		if j, ok := json[name]; ok {
			out.Write(j)
		}
		dataset[name] = out.Bytes()
	}
	return dataset
}

func FileToRequests(file string, processor Processor, opts ...func(*Config)) (map[string]entities.TextToImageRequest, error) {
	p, err := FileToParams(file, processor, opts...)
	if err != nil {
		return nil, err
	}
	return ParseParams(p), nil
}

// FileToParams reads the file and returns the params using a Processor
func FileToParams(file string, processor Processor, opts ...func(*Config)) (Params, error) {
	f, err := FileToBytes(file)
	if err != nil {
		return nil, err
	}
	opts = append(opts, WithFilename(file))
	opts = append(opts, WithBytes(f))
	return processor(opts...)
}

// FileToBytes reads the file and returns the content as a byte slice
func FileToBytes(file string) ([]byte, error) {
	f, err := os.OpenFile(file, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(f)
}

// InputResponse is a struct that holds the input and response training data for LLMs
type InputResponse struct {
	Input    string
	Response []byte
}

// MapParams returns the split params files as a map with the corresponding json for LLM training
func MapParams(processor Processor, opts ...func(*Config)) map[string]InputResponse {
	params, err := processor(opts...)
	if err != nil {
		return nil
	}

	if params == nil {
		return nil
	}

	request := ParseParams(params)
	if request == nil {
		return nil
	}

	var out map[string]InputResponse
	for name, r := range request {
		marshal, err := json.MarshalIndent(map[string]entities.TextToImageRequest{name: r}, "", "  ")
		if err != nil {
			continue
		}
		if out == nil {
			out = make(map[string]InputResponse)
		}
		s := InputResponse{
			Response: marshal,
		}
		if chunk, ok := params[name]; ok {
			if p, ok := chunk[Parameters]; ok {
				s.Input = p
			}
		}
		if s.Input != "" {
			out[name] = s
		}
	}
	return out
}

const empty = `###Instruction:
{example['instruction']}

### Input:
{example['input']}

### Response:`

const commonInstruction = `###Instruction: 
You are a backend API that responds to requests in natural language and outputs a raw JSON object.
Process the following description of an image generated with Stable Diffusion.
Output only a raw JSON response and do not include any comments.
IMPORTANT: Do not include comments, only output the JSON object.
Sometimes there's more than one prompt, so intelligently recognize this.
Keep loras as is <lora:MODELNAME:weight>
Use the following JSON format: 
{"filename": {
"steps": <|steps|>,
"width": <|width|>,
"height": <|height|>,
"seed": <|seed|>,
"n_iter": <|n_iter|>, // also known as batch count
"batch_size": <|batch_size|>,
"prompt": <|prompt|>, // look for positive prompt, keep loras as is, e.g. <lora:MODELNAME:float>
"negative_prompt": <|negative_prompt|>, // look for negative prompt, keep loras as is, e.g. <lora:MODELNAME:float>
"sampler_name": <|sampler_name|>, // default is Euler a
"override_settings": {
  "sd_model_checkpoint": <|sd_model_checkpoint|>, // also known as model
  "sd_checkpoint_hash": <|sd_checkpoint_hash|> // also known as model hash
},
"alwayson_scripts": {
 "ADetailer": { // ADetailer is only an example
   "args": [] // contains an "args" array with any type inside
 }
}, // "script": OBJECTS. Include any additional information here such as CFG Rescale, Controlnet, ADetailer, RP, etc.
"cfg_scale": <|cfg_scale|>, // not to be confused rescale
"comments": {  "description": <|description|>  }, // Find the generator used. Default is Stable Diffusion, ComfyUI, etc.
"denoising_strength": <|denoising_strength|>,
"enable_hr": <|enable_hr|>,
"hr_scale": <|hr_scale|>, // use 2 if not present
"hr_second_pass_steps": <|hr_second_pass_steps|>, // use the same value as steps if not present
"hr_upscaler": <|hr_upscaler|> // default is Latent
}}

`

const completeSample = `###Instruction: 
You are a backend API that responds to requests in natural language and outputs a raw JSON object.
Process the following description of an image generated with Stable Diffusion.
Output only a raw JSON response and do not include any comments.
IMPORTANT: Do not include comments, only output the JSON object.
Sometimes there's more than one prompt, so intelligently recognize this.
Keep loras as is <lora:MODELNAME:weight>
Use the following JSON format: 
{"filename": {
"steps": <|steps|>,
"width": <|width|>,
"height": <|height|>,
"seed": <|seed|>,
"n_iter": <|n_iter|>, // also known as batch count
"batch_size": <|batch_size|>,
"prompt": <|prompt|>, // look for positive prompt, keep loras as is, e.g. <lora:MODELNAME:float>
"negative_prompt": <|negative_prompt|>, // look for negative prompt, keep loras as is, e.g. <lora:MODELNAME:float>
"sampler_name": <|sampler_name|>, // default is Euler a
"override_settings": {
  "sd_model_checkpoint": <|sd_model_checkpoint|>, // also known as model
  "sd_checkpoint_hash": <|sd_checkpoint_hash|> // also known as model hash
},
"alwayson_scripts": {
 "ADetailer": { // ADetailer is only an example
   "args": [] // contains an "args" array with any type inside
 }
}, // "script": OBJECTS. Include any additional information here such as CFG Rescale, Controlnet, ADetailer, RP, etc.
"cfg_scale": <|cfg_scale|>, // not to be confused rescale
"comments": {  "description": <|description|>  }, // Find the generator used. Default is Stable Diffusion, ComfyUI, etc.
"denoising_strength": <|denoising_strength|>,
"enable_hr": <|enable_hr|>,
"hr_scale": <|hr_scale|>, // use 2 if not present
"hr_second_pass_steps": <|hr_second_pass_steps|>, // use the same value as steps if not present
"hr_upscaler": <|hr_upscaler|> // default is Latent
}}

### Input:
{example['input']}

### Response:
{
"filename": {
 "steps": 20,
 "width": 512,
 "height": 512,
 "seed": 1234,
 "n_iter": 1,
 "batch_size": 1,
 "prompt": "<|prompt|>", 
 "negative_prompt": "<|negative_prompt|>", 
 "sampler_name": "<|sampler_name|>",
 "override_settings": {
   "sd_model_checkpoint": "<|sd_model_checkpoint|>", 
   "sd_checkpoint_hash": "<|sd_checkpoint_hash|>" 
 },
 "alwayson_scripts": {
  "ADetailer": { 
    "args": [] 
  }
 }, 
 "cfg_scale": 7, 
 "comments": {  "description": "<|description|>"  }, 
 "denoising_strength": 0.4,
 "enable_hr": true,
 "hr_scale": 2,
 "hr_second_pass_steps": 20, 
 "hr_upscaler": "<|hr_upscaler|>"
 }
}`

package utils

import (
	"bytes"
	"regexp"
)

var (
	Patterns = map[string]*regexp.Regexp{
		"steps":      regexp.MustCompile(`(?i)steps:? (?P<steps>\d+)`),
		"sampler":    regexp.MustCompile(`(?i)sampler:? (?P<sampler>[\w+ ]+)`),
		"cfg":        regexp.MustCompile(`(?i)cfg scale:? (?P<cfg>[\d.]+)`),
		"seed":       regexp.MustCompile(`(?i)seed:? (?P<seed>\d+)`),
		"width":      regexp.MustCompile(`(?i)size:? (?P<width>\d+)x\d+`),
		"height":     regexp.MustCompile(`(?i)size:? \d+x(?P<height>\d+)`),
		"hash":       regexp.MustCompile(`(?i)model hash:? (?P<hash>\w+)`),
		"model":      regexp.MustCompile(`(?i)(?:model|checkpoint) ?[^h]:? (?P<model>[\w-]+)`),
		"denoising":  regexp.MustCompile(`(?i)denoising strength:? (?P<denoising>[\d.]+)`),
		"loraHashes": regexp.MustCompile(loraHashes),
		"tiHashes":   regexp.MustCompile(tiHashes),
		"version":    regexp.MustCompile(`(?i)version:? (?P<version>v[\w.-]+)`),
	}

	allParams = regexp.MustCompile(`\s*(\w[\w \-/]+):\s*("(?:\\.|[^\\"])+"|[^,]*)(?:,|$)`)

	positivePattern = regexp.MustCompile(`(?is)(?:(?:primary |pos(?:itive)? )?prompts?:?)\s*(.+)\s*negative(?: prompt:?)?`)
	negativePattern = regexp.MustCompile(`(?is)(?:(?:neg(?:ative)?)(?: prompts?)?:?)\s*(.+)\s*(?:steps|sampler|model)`)
	negativeEnd     = regexp.MustCompile(`(?is)(?:(?:neg(?:ative)?)(?: prompts?)?:?)\s*(.+)`)
	bbCode          = regexp.MustCompile(`\[\/?[\w=]+\]`)

	extractJson    = regexp.MustCompile(`(?ms){.*}`)
	removeComments = regexp.MustCompile(`(?m)//.*$`)
	fixParentheses = regexp.MustCompile(`\\+([()])`)

	newLineFix       = []byte("\n")
	quoteFix         = []byte(`\"`)
	stringExtraction = regexp.MustCompile(`(?s)(:\s+")(.*?)(",\n)|(?s)(:\s+")(.*?)(".*?\n)`)

	parenthesesReplacement = []byte(`\\$1`)
	newLineReplacement     = []byte(`\n`)
	quoteReplacement       = []byte(`"`)
)

const (
	loraHashes = `(?i)lora hashes:? "(?P<lora>[^"]+)"`
	loraEntry  = `(?i)(?P<key>\w+): (?P<value>\w+)`
	tiHashes   = `(?i)ti hashes:? "(?P<ti>[^"]+)"`
	tiEntry    = `(?i)(?P<key>\w+): (?P<value>\w+)`
)

func RemoveBBCode(s string) string {
	return bbCode.ReplaceAllString(s, "")
}

type ExtractResult map[string]string

func ExtractAll(s string, reg map[string]*regexp.Regexp) ExtractResult {
	var result = make(ExtractResult)

	for key, r := range reg {
		result[key] = Extract(s, r)
	}

	return result
}

// ExtractKeys extracts keys and values from a string using regex.
// Usually only the last line is passed here
//
//	`\s*(\w[\w \-/]+):\s*("(?:\\.|[^\\"])+"|[^,]*)(?:,|$)`
//
// [Key: value], [Key: value], [Key: "key: value, key:value"]
func ExtractKeys(line string) ExtractResult {
	var result = make(ExtractResult)

	matches := allParams.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		result[match[1]] = match[2]
	}

	return result
}

// ExtractDefaultKeys can use a default mapping to store the results to.
// Usually only the last line is passed here
func ExtractDefaultKeys(line string, defaultResults ExtractResult) ExtractResult {
	if defaultResults == nil {
		defaultResults = make(ExtractResult)
	}

	matches := allParams.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		defaultResults[match[1]] = match[2]
	}

	return defaultResults
}

func Extract(s string, r *regexp.Regexp) string {
	match := r.FindStringSubmatch(s)

	for i, name := range match {
		if i != 0 {
			return name
		}
	}

	return ""
}

func ExtractJson(content []byte) []byte {
	// Extracts everything from `{.*}`
	content = extractJson.Find(content)

	// Fix newlines inside strings
	// replace 2nd and 5th capturing group's char(10) newlines inside strings to `\n`
	content = stringExtraction.ReplaceAllFunc(content, func(s []byte) []byte {
		matches := stringExtraction.FindSubmatch(s)
		// replace newlines inside strings with \n
		matches[2] = bytes.ReplaceAll(matches[2], newLineFix, newLineReplacement)
		matches[5] = bytes.ReplaceAll(matches[5], newLineFix, newLineReplacement)
		return bytes.Join(matches[1:], nil)
	})

	// Fix escaped parentheses e.g. `\(text\)` to `\\(text\\)`
	content = fixParentheses.ReplaceAll(content, parenthesesReplacement)
	content = removeComments.ReplaceAll(content, nil)
	return content
}

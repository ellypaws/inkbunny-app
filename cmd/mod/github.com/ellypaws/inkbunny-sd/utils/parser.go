package utils

import "strings"

// ExtractPositiveBackwards extracts the positive prompt by searching backwards from the negative prompt
func ExtractPositiveBackwards(s string) string {
	return SearchFrom(s, "negative prompt",
		"positive prompt",
		"prompt",
		"technical information",
		"[i]",
		"[b]",
	)
}

func ExtractPositiveForward(s string) string {
	return SearchTo(s, "positive prompt",
		"negative prompt",
		"steps",
	)
}

func ExtractNegativeBackwards(s string) string {
	return SearchFrom(s, "positive prompt",
		"negative prompt",
		"prompt",
		"technical information",
		"[i]",
		"[b]",
	)
}

func ExtractNegativeForward(s string) string {
	return SearchTo(s, "negative prompt",
		"positive prompt",
		"steps",
		"sampler",
		"model",
	)
}

func SearchFrom(s, end string, stops ...string) string {
	e := strings.Index(strings.ToLower(s), end)
	if e == -1 {
		return ""
	}

	beforeEnd := s[:e]

	startIndex, foundString := hasLastIndex(strings.ToLower(beforeEnd), stops...)

	if startIndex == -1 {
		return beforeEnd
	}

	return beforeEnd[startIndex+len(foundString):]
}

func hasLastIndex(s string, substr ...string) (int, string) {
	for _, sub := range substr {
		if i := strings.LastIndex(s, sub); i != -1 {
			return i, sub
		}
	}
	return -1, ""
}

func SearchTo(s, start string, stops ...string) string {
	e := strings.Index(strings.ToLower(s), start)
	if e == -1 {
		return ""
	}

	afterStart := s[e:]

	startIndex, _ := hasIndex(strings.ToLower(afterStart), stops...)

	if startIndex == -1 {
		return afterStart
	}

	return afterStart[:startIndex]
}

func hasIndex(s string, substr ...string) (int, string) {
	for _, sub := range substr {
		if i := strings.Index(s, sub); i != -1 {
			return i, sub
		}
	}
	return -1, ""
}

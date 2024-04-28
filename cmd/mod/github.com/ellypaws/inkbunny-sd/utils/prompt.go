package utils

import (
	"strings"
)

func ExtractPositivePrompt(s string) string {
	s = RemoveBBCode(s)
	result := Extract(s, positivePattern)

	if result == "" {
		result = ExtractPositiveBackwards(s)
	}

	if result == "" {
		result = ExtractPositiveForward(s)
	}

	if result == "" {
		result = Extract(s, positiveEnd)
	}

	return trim(result)
}

func ExtractNegativePrompt(s string) string {
	s = RemoveBBCode(s)
	result := Extract(s, negativePattern)

	if result == "" {
		result = ExtractNegativeForward(s)
	}

	if result == "" {
		result = ExtractNegativeBackwards(s)
	}

	if result == "" {
		result = Extract(s, negativeEnd)
	}

	return trim(result)
}

func trim(s string) string {
	return strings.Trim(s, " \n|[]")
}

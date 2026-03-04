package nlp

import (
	"strings"
)

func ExtractJSON(text string) string {
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(text, "}")
	if end == -1 {
		return ""
	}
	return text[start : end+1]
}

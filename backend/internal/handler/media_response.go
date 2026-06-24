package handler

import (
	"mime"
	"path"
	"strings"
	"unicode/utf8"
)

const fallbackMediaPreviewFileName = "task-media"

func inlineMediaContentDisposition(fileName string) string {
	safeName := sanitizeMediaPreviewFileName(fileName)
	disposition := mime.FormatMediaType("inline", map[string]string{"filename": safeName})
	if disposition == "" {
		return `inline; filename="` + fallbackMediaPreviewFileName + `"`
	}
	return disposition
}

func sanitizeMediaPreviewFileName(fileName string) string {
	normalized := strings.TrimSpace(fileName)
	normalized = strings.ReplaceAll(normalized, "\\", "/")
	normalized = path.Base(normalized)
	normalized = strings.TrimSpace(normalized)
	normalized = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, normalized)
	normalized = strings.Trim(strings.TrimSpace(normalized), `"'`)
	if normalized == "" || normalized == "." || normalized == "/" {
		return fallbackMediaPreviewFileName
	}
	const maxRunes = 120
	if utf8.RuneCountInString(normalized) > maxRunes {
		runes := []rune(normalized)
		normalized = string(runes[:maxRunes])
	}
	return normalized
}

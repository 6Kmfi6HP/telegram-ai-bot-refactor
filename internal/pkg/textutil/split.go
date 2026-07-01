package textutil

import (
	"strings"
	"unicode/utf8"
)

// SplitTextSafely splits text by byte size while preserving UTF-8 rune
// boundaries and preferring line or space breaks near the end of each chunk.
func SplitTextSafely(text string, maxBytes int) []string {
	if text == "" {
		return nil
	}
	if maxBytes <= 0 || len(text) <= maxBytes {
		return []string{text}
	}

	var chunks []string
	currentPos := 0
	textBytes := []byte(text)

	for currentPos < len(textBytes) {
		end := currentPos + maxBytes
		if end > len(textBytes) {
			end = len(textBytes)
		}

		if end < len(textBytes) {
			for end > currentPos && !utf8.RuneStart(textBytes[end]) {
				end--
			}

			chunk := string(textBytes[currentPos:end])
			chunkLen := len(chunk)
			if lastNewline := strings.LastIndex(chunk, "\n"); lastNewline > chunkLen*3/4 {
				end = currentPos + lastNewline + 1
			} else if lastSpace := strings.LastIndex(chunk, " "); lastSpace > chunkLen*3/4 {
				end = currentPos + lastSpace + 1
			}
		}

		if end <= currentPos {
			end = currentPos + maxBytes
			if end > len(textBytes) {
				end = len(textBytes)
			}
			for end > currentPos && !utf8.Valid(textBytes[currentPos:end]) {
				end--
			}
			if end <= currentPos {
				end = len(textBytes)
			}
		}

		chunks = append(chunks, string(textBytes[currentPos:end]))
		currentPos = end
	}

	return chunks
}

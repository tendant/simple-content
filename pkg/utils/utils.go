package utils

import (
	"strings"
	"unicode"
)

// sanitizeFilename converts a filename to ASCII, replacing non-ASCII characters
// with their closest ASCII equivalents or removing them if no equivalent exists
func SanitizeFilename(filename string) string {
	if filename == "" {
		return ""
	}

	// Create a new string builder with the same capacity as the input
	var result strings.Builder
	result.Grow(len(filename))

	// Process each rune in the filename
	for _, r := range filename {
		// If the rune is ASCII (0-127), keep it
		if r < 128 && unicode.IsPrint(r) {
			result.WriteRune(r)
		} else {
			// For non-ASCII characters, try to find an ASCII equivalent
			switch {
			case unicode.Is(unicode.Latin, r):
				// For Latin characters, try to strip diacritics
				switch {
				case r >= '\u00c0' && r <= '\u00c5':
					result.WriteRune('A')
				case r >= '\u00e0' && r <= '\u00e5':
					result.WriteRune('a')
				case r >= '\u00c8' && r <= '\u00cb':
					result.WriteRune('E')
				case r >= '\u00e8' && r <= '\u00eb':
					result.WriteRune('e')
				case r >= '\u00cc' && r <= '\u00cf':
					result.WriteRune('I')
				case r >= '\u00ec' && r <= '\u00ef':
					result.WriteRune('i')
				case r >= '\u00d2' && r <= '\u00d6':
					result.WriteRune('O')
				case r >= '\u00f2' && r <= '\u00f6':
					result.WriteRune('o')
				case r >= '\u00d9' && r <= '\u00dc':
					result.WriteRune('U')
				case r >= '\u00f9' && r <= '\u00fc':
					result.WriteRune('u')
				case r == '\u00c7':
					result.WriteRune('C')
				case r == '\u00e7':
					result.WriteRune('c')
				case r == '\u00d1':
					result.WriteRune('N')
				case r == '\u00f1':
					result.WriteRune('n')
				default:
					// Replace other non-ASCII Latin characters with underscore
					result.WriteRune('-')
				}
			default:
				// Replace all other non-ASCII characters with underscore
				result.WriteRune('-')
			}
		}
	}

	return result.String()
}

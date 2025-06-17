package utils

import (
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "ascii only",
			input:    "simple-file-name.txt",
			expected: "simple-file-name.txt",
		},
		{
			name:     "with spaces",
			input:    "file with spaces.pdf",
			expected: "file with spaces.pdf",
		},
		{
			name:     "with special characters",
			input:    "file!@#$%^&*().zip",
			expected: "file!@#$%^&*().zip",
		},
		{
			name:     "with latin accents",
			input:    "résumé.pdf",
			expected: "resume.pdf",
		},
		{
			name:     "with latin accents uppercase",
			input:    "RÉSUMÉ.PDF",
			expected: "RESUME.PDF",
		},
		{
			name:     "with mixed latin accents",
			input:    "Café Ñandú.doc",
			expected: "Cafe Nandu.doc",
		},
		{
			name:     "with emojis",
			input:    "document📄.pdf",
			expected: "document-.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

package editor

import "testing"

func TestAddTab(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"empty line": {
			input:    "",
			expected: "\t",
		},
		"simple text": {
			input:    "test line",
			expected: "\ttest line",
		},
		"already indented text": {
			input:    "\tindented",
			expected: "\t\tindented",
		},
		"line with spaces": {
			input:    "    spaced content",
			expected: "\t    spaced content",
		},
		"special characters": {
			input:    "!@#$%^&*()",
			expected: "\t!@#$%^&*()",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := AddTab(tt.input)
			if result != tt.expected {
				t.Errorf("AddTab(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRemoveTab(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"empty line": {
			input:    "",
			expected: "",
		},
		"line with 4 spaces": {
			input:    "    test line",
			expected: "test line",
		},
		"line with less than 4 spaces": {
			input:    "   test line",
			expected: "   test line",
		},
		"line with more than 4 spaces": {
			input:    "     test line",
			expected: " test line",
		},
		"line with no leading spaces": {
			input:    "test line",
			expected: "test line",
		},
		"line with mixed spaces and content": {
			input:    "    test    line",
			expected: "test    line",
		},
		"only spaces": {
			input:    "    ",
			expected: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := RemoveTab(tt.input)
			if result != tt.expected {
				t.Errorf("RemoveTab(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

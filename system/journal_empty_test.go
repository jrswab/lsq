package system

import (
	"testing"
)

func TestIsJournalEmpty(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "zero-length input",
			content:  []byte(""),
			expected: true,
		},
		{
			name:     "only newlines",
			content:  []byte("\n\n\n"),
			expected: true,
		},
		{
			name:     "only whitespace",
			content:  []byte("   \t\n  \n"),
			expected: true,
		},
		{
			name:     "only whitespace and tabs",
			content:  []byte("\t  \t  \n"),
			expected: true,
		},
		{
			name:     "single placeholder with space",
			content:  []byte("- \n"),
			expected: true,
		},
		{
			name:     "single placeholder without space",
			content:  []byte("-\n"),
			expected: true,
		},
		{
			name:     "multiple placeholders",
			content:  []byte("- \n\n- \n"),
			expected: true,
		},
		{
			name:     "placeholder with trailing whitespace",
			content:  []byte("-   \n"),
			expected: true,
		},
		{
			name:     "blank lines and placeholders only",
			content:  []byte("\n- \n\n-   \n\n"),
			expected: true,
		},
		{
			name:     "actual journal entry",
			content:  []byte("- actual entry\n"),
			expected: false,
		},
		{
			name:     "actual entry with no trailing newline",
			content:  []byte("- actual entry"),
			expected: false,
		},
		{
			name:     "mix of placeholders and real content",
			content:  []byte("- \n\n- actual entry\n"),
			expected: false,
		},
		{
			name:     "real content with blank lines",
			content:  []byte("\n\n- real content here\n\n"),
			expected: false,
		},
		{
			name:     "placeholder then real then blank",
			content:  []byte("- \n- real line\n\n"),
			expected: false,
		},
		{
			name:     "unicode content",
			content:  []byte("- 测试 Test テスト\n"),
			expected: false,
		},
		{
			name:     "unicode only on line",
			content:  []byte("- 日本語\n"),
			expected: false,
		},
		{
			name:     "empty bullet followed by unicode",
			content:  []byte("- \n- 日本語\n"),
			expected: false,
		},
		{
			name:     "dash at start with content",
			content:  []byte("- not empty\n"),
			expected: false,
		},
		{
			name:     "dash followed by tab then content",
			content:  []byte("-\tcontent\n"),
			expected: false,
		},
		{
			name:     "multiple dashes in content",
			content:  []byte("- item - with - dashes\n"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isJournalEmpty(tt.content)
			if result != tt.expected {
				t.Errorf("isJournalEmpty(%q) = %v, want %v",
					tt.content, result, tt.expected)
			}
		})
	}
}

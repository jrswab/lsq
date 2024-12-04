package tui

import "strings"

func manipulateText(m *tuiModel, f func(string) string) {
	// Get current content and line number
	content := m.textarea.Value()
	lineNum := m.textarea.Line()

	lines := strings.Split(content, "\n")
	if lineNum < len(lines) {
		lines[lineNum] = f(lines[lineNum])

		newContent := strings.Join(lines, "\n")
		m.textarea.SetValue(newContent)
	}
}

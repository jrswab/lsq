package tui

import (
	"fmt"
	"strings"
)

func (m tuiModel) searchModalView() string {
	if !m.search.active {
		return ""
	}

	// Create modal border and header
	width := 60
	height := 15
	modal := "┌" + strings.Repeat("─", width-2) + "┐\n"

	// Add search input area
	searchPrompt := fmt.Sprintf("Search: %s", m.search.query)
	modal += "│ " + searchPrompt + strings.Repeat(" ", width-len(searchPrompt)-3) + "│\n"
	modal += "│" + strings.Repeat("─", width-2) + "│\n"

	// Add results or "no results" message
	if len(m.search.results) == 0 {
		noResults := "No results found"
		padding := (width - len(noResults) - 2) / 2
		modal += "│" + strings.Repeat(" ", padding) + noResults +
			strings.Repeat(" ", width-len(noResults)-padding-2) + "│\n"
	} else {
		// Show results with selection highlight
		for i, result := range m.search.results {
			if i >= 10 {
				break
			}

			displayResult := result
			if len(displayResult) > width-6 {
				displayResult = displayResult[:width-9] + "..."
			}

			if i == m.search.selectedIndex {
				displayResult = "> " + displayResult
			} else {
				displayResult = "  " + displayResult
			}

			modal += "│ " + displayResult +
				strings.Repeat(" ", width-len(displayResult)-3) + "│\n"
		}
	}

	// Fill remaining space
	remainingLines := height - 5 - min(len(m.search.results), 10)
	for i := 0; i < remainingLines; i++ {
		modal += "│" + strings.Repeat(" ", width-2) + "│\n"
	}

	// Add controls footer
	modal += "│" + strings.Repeat("─", width-2) + "│\n"
	controls := "↑↓:select  Enter:open  Esc:close"
	padding := (width - len(controls) - 2) / 2
	modal += "│" + strings.Repeat(" ", padding) + controls +
		strings.Repeat(" ", width-len(controls)-padding-2) + "│\n"

	// Close modal
	modal += "└" + strings.Repeat("─", width-2) + "┘"

	return modal
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

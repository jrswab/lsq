package tui

import (
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jrswab/lsq/editor"
	"github.com/jrswab/lsq/todo"
)

func (m tuiModel) key(msg tea.KeyMsg) (tuiModel, func() tea.Msg) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyCtrlS:
		content := m.textarea.Value()
		err := os.WriteFile(m.filepath, []byte(content), 0644)
		if err != nil {
			m.statusMsg = "Error saving file!"
			return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
				return statusMsg{}
			})
		}

		m.statusMsg = "File saved successfully!"
		return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
			return statusMsg{}
		})

	case tea.KeyTab:
		manipulateText(&m, editor.AddTab)

	case tea.KeyShiftTab:
		manipulateText(&m, editor.RemoveTab)

	// Cycle through TODO states:
	case tea.KeyCtrlT:
		manipulateText(&m, todo.CycleState)

	case tea.KeyCtrlP:
		manipulateText(&m, todo.CyclePriority)

	case tea.KeyCtrlF:
		m.search.active = !m.search.active
		m.search.query = "" // Reset query when opening search
		m.search.results = nil
		m.search.selectedIndex = 0
		return m, nil

	default:
		// If search is active, handle search-specific keys
		if m.search.active {
			switch msg.Type {
			case tea.KeyEsc:
				m.search.active = false
				return m, nil

			case tea.KeyUp:
				if m.search.selectedIndex > 0 {
					m.search.selectedIndex--
				}
				return m, nil

			case tea.KeyDown:
				if m.search.selectedIndex < len(m.search.results)-1 {
					m.search.selectedIndex++
				}
				return m, nil

			case tea.KeyEnter:
				if len(m.search.results) > 0 {
					// Save current file first
					content := m.textarea.Value()
					err := os.WriteFile(m.filepath, []byte(content), 0644)
					if err != nil {
						m.statusMsg = "Error saving current file!"
						return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
							return statusMsg{}
						})
					}

					// Get selected file path and load it
					selectedFile := m.search.results[m.search.selectedIndex]
					newPath := filepath.Join(filepath.Dir(m.filepath), "..", "pages", selectedFile)

					// Read new file content
					searchContent, err := os.ReadFile(newPath)
					if err != nil {
						m.statusMsg = "Error opening selected file!"
						return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
							return statusMsg{}
						})
					}

					// Update textarea with new content
					m.textarea.SetValue(string(searchContent))
					m.filepath = newPath
					m.search.active = false
					m.statusMsg = "File loaded successfully!"
					return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
						return statusMsg{}
					})
				}

				return m, nil

			// Handle typing for search input
			default:
				if msg.Type == tea.KeyRunes {
					m.search.query += string(msg.Runes)

					// Perform search with updated query
					if m.search.query != "" {
						m.search.results = m.trie.Search(m.search.query)
						// Reset selection when results change
						m.search.selectedIndex = 0
					} else {
						m.search.results = nil
					}
					return m, nil
				}
				if msg.Type == tea.KeyBackspace {
					if len(m.search.query) > 0 {
						m.search.query = m.search.query[:len(m.search.query)-1]

						// Perform search with updated query
						if m.search.query != "" {
							m.search.results = m.trie.Search(m.search.query)
							// Reset selection when results change
							m.search.selectedIndex = 0
						} else {
							m.search.results = nil
						}
					}
					return m, nil
				}
			}
		} else {
			// When search is not active, pass keystrokes to textarea
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

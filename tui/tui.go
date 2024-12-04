package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jrswab/lsq/config"
	"github.com/jrswab/lsq/editor"
	"github.com/jrswab/lsq/todo"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type tuiModel struct {
	//viewport viewport.Model
	textarea  textarea.Model
	config    *config.Config
	filepath  string
	statusMsg string
}

func InitialModel(cfg *config.Config, fp string) tuiModel {
	// Read file content for TUI
	content, err := os.ReadFile(fp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading journal file: %v\n", err)
		os.Exit(1)
	}

	var ta = textarea.New()
	ta.SetValue(string(content))
	ta.Focus()
	ta.CharLimit = -1

	return tuiModel{
		textarea: ta,
		config:   cfg,
		filepath: fp,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return textarea.Blink
}

type statusMsg struct{}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case statusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
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
		}

	case tea.WindowSizeMsg:
		m.textarea.SetWidth(msg.Width - 2)
		m.textarea.SetHeight(msg.Height - 2)
	}

	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m tuiModel) View() string {
	var footer = "^S save, ^C quit"

	if m.statusMsg != "" {
		footer = m.statusMsg
	}

	return fmt.Sprintf(
		"LSQ TUI - %s\n%s\n%s",
		filepath.Base(m.filepath),
		m.textarea.View(),
		footer,
	)
}

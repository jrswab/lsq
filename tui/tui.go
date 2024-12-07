package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jrswab/lsq/config"
	"github.com/jrswab/lsq/trie"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// New search state struct
type searchState struct {
	active        bool
	query         string
	results       []string
	selectedIndex int
}

// Message types for search operations
type searchToggleMsg struct{}
type searchUpdateMsg struct {
	query string
}
type searchSelectMsg struct {
	index int
}

type tuiModel struct {
	textarea  textarea.Model
	config    *config.Config
	filepath  string
	statusMsg string
	search    searchState
	trie      *trie.Trie
}

func InitialModel(cfg *config.Config, fp string, t *trie.Trie) tuiModel {
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
		trie:     t,
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
		return m.key(msg)

	case tea.WindowSizeMsg:
		m.textarea.SetWidth(msg.Width - 2)
		m.textarea.SetHeight(msg.Height - 2)
	}

	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m tuiModel) View() string {
	var footer = "^S save, ^C quit, ^F search"

	if m.statusMsg != "" {
		footer = m.statusMsg
	}

	baseView := fmt.Sprintf(
		"LSQ TUI - %s\n%s\n%s",
		filepath.Base(m.filepath),
		m.textarea.View(),
		footer,
	)

	if m.search.active {
		// Center the modal
		return lipgloss.Place(
			m.textarea.Width(),
			m.textarea.Height(),
			lipgloss.Center,
			lipgloss.Center,
			m.searchModalView(),
		)
	}

	return baseView
}

// Package todo provides functionality for cycling through TODO states and priorities
// in a Logseq-compatible format.
package todo

import (
	"fmt"
	"strings"
)

// States represents the valid TODO states in order of cycling
var States = []string{"TODO", "DOING", "DONE"}

// Priorities represents the valid priority levels in order of cycling
var Priorities = []string{"[#A]", "[#B]", "[#C]"}

// CycleState takes a line of text and returns the same line with the next TODO state.
// If no state exists, it adds "TODO" at the start of the line.
func CycleState(line string) string {
	// Extract indentation and content:
	var indent, content, found = strings.Cut(line, "- ")

	if !found {
		return line
	}

	// Do nothing on an empty line:
	if strings.EqualFold(content, "") {
		return line
	}

	// Find current todo state:
	var currentState string
	for _, state := range States {
		if strings.HasPrefix(content, state) {
			currentState = state
			content = strings.TrimPrefix(content, state)
			break
		}
	}

	switch currentState {
	case States[0]: // TODO
		return fmt.Sprintf("%s- DOING%s", indent, content)
	case States[1]: // DOING
		return fmt.Sprintf("%s- DONE%s", indent, content)
	case States[2]: // DONE
		return fmt.Sprintf("%s-%s", indent, content)
	default:
		return fmt.Sprintf("%s- TODO %s", indent, content)
	}
}

// CyclePriority takes a line of text and returns the same line with the next priority level.
// Only adds/cycles priority if the line starts with a TODO state.
func CyclePriority(line string) string {
	// Extract indentation and content:
	var indent, content, _ = strings.Cut(line, "- ")

	// Do nothing on an empty line:
	if strings.EqualFold(content, "") {
		return line
	}

	// Check if line starts with a TODO state
	var (
		hasState    bool
		statePrefix string
	)

	for _, state := range States {
		if strings.HasPrefix(content, fmt.Sprintf("%s ", state)) {
			hasState = true
			statePrefix = state

			content = strings.TrimPrefix(content, fmt.Sprintf("%s ", state))
			break
		}
	}

	// If no TODO state, return original line unchanged
	if !hasState {
		return line
	}

	// Find current priority
	var currentPriority string
	for _, priority := range Priorities {
		if strings.HasPrefix(content, fmt.Sprintf("%s ", priority)) {
			currentPriority = priority

			content = strings.TrimPrefix(content, fmt.Sprintf("%s ", priority))
			break
		}
	}

	switch currentPriority {
	case Priorities[0]: // [#A]
		return fmt.Sprintf("%s- %s [#B] %s", indent, statePrefix, content)
	case Priorities[1]: // [#B]
		return fmt.Sprintf("%s- %s [#C] %s", indent, statePrefix, content)
	case Priorities[2]: // [#C]
		return fmt.Sprintf("%s- %s %s", indent, statePrefix, content)
	default:
		// When no TODO state is found the function returns the original line
		// before checking for a current priority.
		return fmt.Sprintf("%s- %s [#A] %s", indent, statePrefix, content)
	}
}

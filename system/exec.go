package system

import (
	"fmt"
	"os"
	"os/exec"
)

func LoadEditor(editor, path string) {
	// Get editor from environment
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}

	// if still blank, use nano
	if editor == "" {
		fmt.Println("$EDITOR is blank, using Vim.")
		editor = "vim"
	}

	// Open file in editor
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error opening editor: %v\n", err)
		os.Exit(1)
	}
}

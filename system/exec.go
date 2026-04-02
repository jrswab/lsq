package system

import (
	"fmt"
	"io"
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

// nopCloser wraps an io.Writer with a no-op Close method
type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

// LoadPager returns a writer that writes to a pager (less/more), or stdout if no pager is available.
// The caller must call the returned cleanup function after writing all content.
func LoadPager() (io.WriteCloser, func(), error) {
	// Resolve pager command
	pager := os.Getenv("PAGER")

	if pager == "" {
		if path, err := exec.LookPath("less"); err == nil {
			pager = path
		} else if path, err := exec.LookPath("more"); err == nil {
			pager = path
		}
	}

	// If no pager found, fall back to stdout
	if pager == "" {
		return nopCloser{os.Stdout}, func() {}, nil
	}

	// Create pipe for pager subprocess
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}

	// Setup pager command
	cmd := exec.Command(pager)
	cmd.Stdin = readPipe
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the pager
	if err := cmd.Start(); err != nil {
		readPipe.Close()
		writePipe.Close()
		return nil, nil, err
	}

	// Close read end in parent process (pager has it now)
	readPipe.Close()

	// Create cleanup function
	cleanup := func() {
		// Wait for pager to finish (ignore non-zero exit)
		cmd.Wait()
	}

	return writePipe, cleanup, nil
}

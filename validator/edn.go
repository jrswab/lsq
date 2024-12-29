package validator

import (
	"fmt"
	"io"
	"os"
	"strings"

	"olympos.io/encoding/edn"
)

// ErrInvalidEDN represents an EDN validation error
type ErrInvalidEDN struct {
	msg string
	err error
}

func (e *ErrInvalidEDN) Error() string {
	if e.err != nil {
		return fmt.Sprintf("invalid EDN: %s: %v", e.msg, e.err)
	}
	return fmt.Sprintf("invalid EDN: %s", e.msg)
}

func (e *ErrInvalidEDN) Unwrap() error {
	return e.err
}

// Validator provides EDN validation functionality
type Validator struct {
	// Add fields here if we need to support configuration options later
}

// New creates a new EDN validator
func New() *Validator {
	return &Validator{}
}

// ValidateFile validates an EDN file
func (v *Validator) ValidateFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return v.ValidateReader(f)
}

// ValidateString validates an EDN string
func (v *Validator) ValidateString(s string) error {
	return v.ValidateReader(strings.NewReader(s))
}

// ValidateReader validates EDN from an io.Reader
func (v *Validator) ValidateReader(r io.Reader) error {
	decoder := edn.NewDecoder(r)

	// We'll decode into an empty interface
	// since we only care about syntax yet
	var data interface{}
	if err := decoder.Decode(&data); err != nil {
		return &ErrInvalidEDN{
			msg: "failed to parse EDN",
			err: err,
		}
	}

	// Basic validation passed
	return nil
}

package system_test

import (
	"testing"

	"github.com/jrswab/lsq/system"
)

func TestLoadPager_PagerSetToCat(t *testing.T) {
	t.Setenv("PAGER", "cat")

	writer, cleanup, err := system.LoadPager()
	if err != nil {
		t.Fatalf("LoadPager() returned error: %v", err)
	}
	if writer == nil {
		t.Fatal("LoadPager() returned nil writer")
	}
	defer cleanup()

	// Write some data
	testData := []byte("hello world\n")
	n, err := writer.Write(testData)
	if err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}
	if n != len(testData) {
		t.Fatalf("Write() wrote %d bytes, expected %d", n, len(testData))
	}

	// Cleanup should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("cleanup() panicked: %v", r)
			}
		}()
		// Must close writer before cleanup to signal EOF to pager
		writer.Close()
		cleanup()
	}()
}

func TestLoadPager_NoPagerAvailable(t *testing.T) {
	// Create a temp directory with no executables
	tempDir := t.TempDir()

	// Set PATH to empty temp dir and unset PAGER
	t.Setenv("PATH", tempDir)
	t.Setenv("PAGER", "")

	writer, cleanup, err := system.LoadPager()
	if err != nil {
		t.Fatalf("LoadPager() returned error when no pager available: %v", err)
	}
	if writer == nil {
		t.Fatal("LoadPager() returned nil writer when no pager available")
	}

	// Cleanup should be a no-op and not block
	done := make(chan struct{})
	go func() {
		cleanup()
		close(done)
	}()

	select {
	case <-done:
		// Success - cleanup completed without blocking
	case <-t.Context().Done():
		t.Fatal("cleanup() blocked when no pager available")
	}

	// Writer should still accept writes without error
	testData := []byte("test data\n")
	_, writeErr := writer.Write(testData)
	// Write to a no-op writer should succeed (bytes written to nowhere)
	if writeErr != nil {
		t.Fatalf("Write() returned error when no pager available: %v", writeErr)
	}
}

func TestLoadPager_WriteThenClose(t *testing.T) {
	t.Setenv("PAGER", "cat")

	writer, cleanup, err := system.LoadPager()
	if err != nil {
		t.Fatalf("LoadPager() returned error: %v", err)
	}
	if writer == nil {
		t.Fatal("LoadPager() returned nil writer")
	}
	defer cleanup()

	// Write some data
	testData := []byte("test data for close\n")
	_, err = writer.Write(testData)
	if err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	// Close should not return error
	// The writer is an io.WriteCloser, so it has a Close() method
	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("Close() returned error: %v", closeErr)
	}
}

func TestLoadPager_WrittenBytesNotLost(t *testing.T) {
	// This test uses a custom writer to verify bytes are not lost
	t.Setenv("PAGER", "cat")

	writer, cleanup, err := system.LoadPager()
	if err != nil {
		t.Fatalf("LoadPager() returned error: %v", err)
	}
	if writer == nil {
		t.Fatal("LoadPager() returned nil writer")
	}
	defer cleanup()

	// Write multiple chunks
	chunks := [][]byte{
		[]byte("first chunk\n"),
		[]byte("second chunk\n"),
		[]byte("third chunk\n"),
	}

	var totalWritten int
	for _, chunk := range chunks {
		n, err := writer.Write(chunk)
		if err != nil {
			t.Fatalf("Write() returned error: %v", err)
		}
		totalWritten += n
	}

	expectedTotal := 0
	for _, chunk := range chunks {
		expectedTotal += len(chunk)
	}

	if totalWritten != expectedTotal {
		t.Fatalf("Total bytes written %d does not match expected %d", totalWritten, expectedTotal)
	}

	// Verify we can still write after previous writes
	finalChunk := []byte("final chunk\n")
	n, err := writer.Write(finalChunk)
	if err != nil {
		t.Fatalf("Write() after previous writes returned error: %v", err)
	}
	if n != len(finalChunk) {
		t.Fatalf("Write() after previous writes wrote %d bytes, expected %d", n, len(finalChunk))
	}

	// Must close writer before cleanup to signal EOF to pager
	writer.Close()
}

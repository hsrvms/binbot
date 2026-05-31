package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLineRotator(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "bot.log")

	rotator, err := NewLineRotator(logFile, 5)
	if err != nil {
		t.Fatalf("Failed to initialize rotator: %v", err)
	}

	payload := []byte("Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\n")
	rotator.Write(payload)

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files (active + backup), got %d", len(files))
	}
}

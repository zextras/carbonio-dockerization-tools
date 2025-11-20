package embedded

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// WorkDir is the name of the working directory created next to the binary
const WorkDir = "carbonio-docker-workdir"

// Extractor handles extraction and validation of embedded files
type Extractor struct {
	binaryPath string
	workDir    string
}

// NewExtractor creates a new extractor instance
func NewExtractor() (*Extractor, error) {
	// Get the directory where the binary is located
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	binaryDir := filepath.Dir(exePath)
	workDir := filepath.Join(binaryDir, WorkDir)

	return &Extractor{
		binaryPath: binaryDir,
		workDir:    workDir,
	}, nil
}

// GetWorkDir returns the path to the working directory
func (e *Extractor) GetWorkDir() string {
	return e.workDir
}

// EnsureExtracted ensures files are extracted fresh every time
func (e *Extractor) EnsureExtracted() error {
	log.Println("Ensuring fresh workdir extraction...")

	// Check if workdir exists
	if _, err := os.Stat(e.workDir); err == nil {
		// Workdir exists - remove it completely for fresh extraction
		log.Printf("Removing existing workdir: %s", e.workDir)
		if err := os.RemoveAll(e.workDir); err != nil {
			return fmt.Errorf("failed to remove existing workdir: %w", err)
		}
		log.Println("Existing workdir removed successfully")
	} else if !os.IsNotExist(err) {
		// Some other error occurred
		return fmt.Errorf("failed to check workdir: %w", err)
	}

	// Always extract everything fresh
	log.Println("Extracting fresh workdir...")
	return e.extractAll()
}

// extractAll extracts all embedded files to the working directory
func (e *Extractor) extractAll() error {
	log.Printf("Creating workdir: %s", e.workDir)
	if err := os.MkdirAll(e.workDir, 0755); err != nil {
		return fmt.Errorf("failed to create workdir: %w", err)
	}

	fileCount := 0
	dirCount := 0

	// Walk through embedded filesystem
	err := fs.WalkDir(EmbeddedFiles, "carbonio-base-dockerization", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip root directory
		if path == "carbonio-base-dockerization" {
			return nil
		}

		// Remove prefix "carbonio-base-dockerization/" from path
		relPath := strings.TrimPrefix(path, "carbonio-base-dockerization/")

		// Normalize path separators for cross-platform
		normalizedPath := filepath.FromSlash(relPath)
		targetPath := filepath.Join(e.workDir, normalizedPath)

		if d.IsDir() {
			dirCount++
			log.Printf("Creating directory: %s", relPath)
			return os.MkdirAll(targetPath, 0755)
		}

		// Extract file
		data, err := EmbeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent dir for %s: %w", targetPath, err)
		}

		// Write file with 0755 permissions (executable)
		if err := os.WriteFile(targetPath, data, 0755); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		fileCount++
		log.Printf("Extracted file: %s (%d bytes)", relPath, len(data))

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to extract files: %w", err)
	}

	log.Printf("Extraction complete: %d directories, %d files", dirCount, fileCount)
	return nil
}

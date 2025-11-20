package embedded

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
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

// EnsureExtracted ensures files are extracted and up-to-date
func (e *Extractor) EnsureExtracted() error {
	// Check if workdir exists
	if _, err := os.Stat(e.workDir); os.IsNotExist(err) {
		// First run - extract everything
		return e.extractAll()
	}

	// Workdir exists - check if embedded files have changed
	needsUpdate, err := e.needsUpdate()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if needsUpdate {
		// Remove old directory and re-extract
		if err := os.RemoveAll(e.workDir); err != nil {
			return fmt.Errorf("failed to remove old workdir: %w", err)
		}
		return e.extractAll()
	}

	// Files are up-to-date, reuse existing directory
	return nil
}

// extractAll extracts all embedded files to the working directory
func (e *Extractor) extractAll() error {
	if err := os.MkdirAll(e.workDir, 0755); err != nil {
		return fmt.Errorf("failed to create workdir: %w", err)
	}

	// Walk through embedded filesystem
	// Il path base è "carbonio-base-dockerization" nell'embed
	err := fs.WalkDir(EmbeddedFiles, "carbonio-base-dockerization", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip root directory
		if path == "carbonio-base-dockerization" {
			return nil
		}

		// Rimuovi il prefisso "carbonio-base-dockerization/" dal path
		relPath := strings.TrimPrefix(path, "carbonio-base-dockerization/")

		// Normalize path separators for cross-platform
		normalizedPath := filepath.FromSlash(relPath)
		targetPath := filepath.Join(e.workDir, normalizedPath)

		if d.IsDir() {
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

		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to extract files: %w", err)
	}

	// Write hash file to track version
	if err := e.writeHashFile(); err != nil {
		return fmt.Errorf("failed to write hash file: %w", err)
	}

	return nil
}

// needsUpdate checks if embedded files have changed since last extraction
func (e *Extractor) needsUpdate() (bool, error) {
	hashFile := filepath.Join(e.workDir, ".carbonio-cli-hash")

	// Read existing hash
	existingHash, err := os.ReadFile(hashFile)
	if err != nil {
		// Hash file doesn't exist, needs update
		return true, nil
	}

	// Calculate current embedded files hash
	currentHash, err := e.calculateEmbeddedHash()
	if err != nil {
		return false, err
	}

	return string(existingHash) != currentHash, nil
}

// calculateEmbeddedHash calculates a hash of all embedded files
func (e *Extractor) calculateEmbeddedHash() (string, error) {
	hash := sha256.New()

	err := fs.WalkDir(EmbeddedFiles, "carbonio-base-dockerization", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		data, err := EmbeddedFiles.ReadFile(path)
		if err != nil {
			return err
		}

		// Include path and content in hash
		hash.Write([]byte(path))
		hash.Write(data)

		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// writeHashFile writes the current hash to the workdir
func (e *Extractor) writeHashFile() error {
	hash, err := e.calculateEmbeddedHash()
	if err != nil {
		return err
	}

	hashFile := filepath.Join(e.workDir, ".carbonio-cli-hash")
	return os.WriteFile(hashFile, []byte(hash), 0644)
}

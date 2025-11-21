package embedded

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const WorkDir = "carbonio-docker-workdir"

type Extractor struct {
	binaryPath string
	workDir    string
}

func NewExtractor() (*Extractor, error) {
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
func (e *Extractor) GetWorkDir() string {
	return e.workDir
}
func (e *Extractor) EnsureExtracted() error {
	log.Println("Ensuring fresh workdir extraction...")
	if _, err := os.Stat(e.workDir); err == nil {
		log.Printf("Removing existing workdir: %s", e.workDir)
		if err := os.RemoveAll(e.workDir); err != nil {
			return fmt.Errorf("failed to remove existing workdir: %w", err)
		}
		log.Println("Existing workdir removed successfully")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check workdir: %w", err)
	}
	log.Println("Extracting fresh workdir...")
	return e.extractAll()
}
func (e *Extractor) extractAll() error {
	log.Printf("Creating workdir: %s", e.workDir)
	if err := os.MkdirAll(e.workDir, 0755); err != nil {
		return fmt.Errorf("failed to create workdir: %w", err)
	}
	fileCount := 0
	dirCount := 0
	err := fs.WalkDir(EmbeddedFiles, "carbonio-base-dockerization", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "carbonio-base-dockerization" {
			return nil
		}
		relPath := strings.TrimPrefix(path, "carbonio-base-dockerization/")
		normalizedPath := filepath.FromSlash(relPath)
		targetPath := filepath.Join(e.workDir, normalizedPath)
		if d.IsDir() {
			dirCount++
			log.Printf("Creating directory: %s", relPath)
			return os.MkdirAll(targetPath, 0755)
		}
		data, err := EmbeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent dir for %s: %w", targetPath, err)
		}
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

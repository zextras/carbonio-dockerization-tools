package embedded

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	AppName = "carbonio-dockerization-tools"
	WorkDir = "workdir"
)

type Extractor struct {
	baseDir string
	workDir string
}

func NewExtractor() (*Extractor, error) {
	baseDir, err := getAppCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine cache directory: %w", err)
	}
	workDir := filepath.Join(baseDir, WorkDir)
	return &Extractor{
		baseDir: baseDir,
		workDir: workDir,
	}, nil
}

// getAppCacheDir returns the application's cache directory.
// Uses os.UserCacheDir() which returns:
//   - Linux: ~/.cache/carbonio-dockerization-tools
//   - macOS: ~/Library/Caches/carbonio-dockerization-tools
//   - Windows: %LocalAppData%\carbonio-dockerization-tools
func getAppCacheDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		// Fallback to home directory if cache dir is not available
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", fmt.Errorf("failed to get cache dir (%v) and home dir (%v)", err, homeErr)
		}
		log.Printf("Warning: using home directory as fallback (cache dir unavailable: %v)", err)
		return filepath.Join(homeDir, "."+AppName), nil
	}
	return filepath.Join(cacheDir, AppName), nil
}
func (e *Extractor) GetWorkDir() string {
	return e.workDir
}
func (e *Extractor) EnsureExtracted() error {
	log.Println("Ensuring fresh workdir extraction...")
	if _, err := os.Stat(e.workDir); err == nil {
		log.Printf("Removing existing workdir: %s", e.workDir)
		if err := os.RemoveAll(e.workDir); err != nil {
			log.Printf("Normal removal failed: %v, trying docker cleanup...", err)
			if err := e.dockerCleanup(); err != nil {
				log.Printf("Docker cleanup failed: %v", err)
				return fmt.Errorf("failed to remove existing workdir (try 'sudo rm -rf %s'): %w", e.workDir, err)
			}
			if err := os.RemoveAll(e.workDir); err != nil {
				return fmt.Errorf("failed to remove workdir after docker cleanup: %w", err)
			}
		}
		log.Println("Existing workdir removed successfully")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check workdir: %w", err)
	}
	log.Println("Extracting fresh workdir...")
	return e.extractAll()
}

func (e *Extractor) dockerCleanup() error {
	log.Printf("Using docker to clean up files in: %s", e.workDir)
	cmd := exec.Command("docker", "run", "--rm", "-v", e.workDir+":/cleanup", "busybox", "sh", "-c", "rm -rf /cleanup/* /cleanup/.[!.]* 2>/dev/null || true")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Docker cleanup command output: %s", string(output))
		return fmt.Errorf("docker cleanup failed: %w", err)
	}
	log.Println("Docker cleanup completed successfully")
	return nil
}
func (e *Extractor) extractAll() error {
	log.Printf("Creating workdir: %s", e.workDir)
	if err := os.MkdirAll(e.workDir, 0755); err != nil {
		return fmt.Errorf("failed to create workdir: %w", err)
	}
	fileCount := 0
	dirCount := 0
	err := fs.WalkDir(EmbeddedFiles, "carbonio-dockerization", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "carbonio-dockerization" {
			return nil
		}
		relPath := strings.TrimPrefix(path, "carbonio-dockerization/")
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

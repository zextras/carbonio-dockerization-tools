package logutil

import (
	"os"
)

const maxLogSize = 10 * 1024 * 1024 // 10 MB

// TrimLogFile trims the log file to its last half if it exceeds maxLogSize.
// Should be called before opening the file for append.
func TrimLogFile(path string) {
	info, err := os.Stat(path)
	if err != nil || info.Size() <= maxLogSize {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// Keep the second half
	half := len(data) / 2
	// Advance to the next newline to avoid a partial first line
	for half < len(data) && data[half] != '\n' {
		half++
	}
	if half < len(data) {
		half++
	}

	os.WriteFile(path, data[half:], 0644)
}

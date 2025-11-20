package embedded

import "embed"

// EmbeddedFiles contiene l'intera cartella carbonio-base-dockerization
// Che si trova nella stessa directory di questo file
//
//go:embed all:carbonio-base-dockerization
var EmbeddedFiles embed.FS

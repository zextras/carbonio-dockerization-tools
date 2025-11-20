package embedded

import "embed"

// Embed specific docker-compose files
//
//go:embed ../../docker-compose.yaml
var ComposeYAML []byte

//go:embed ../../docker-compose-advanced.yaml
var ComposeAdvancedYAML []byte

//go:embed ../../composed-ui/Dockerfile
var ComposedUIDockerfile []byte

// For extraction, we'll embed the entire directories needed
//
//go:embed ../../composed-ui
var ComposedUIDir embed.FS

//go:embed ../../postgres
var PostgresDir embed.FS

//go:embed ../../wiremock-consul
var WiremockConsulDir embed.FS

//go:embed ../../carbonio-catalog
var CarbonioCatalogDir embed.FS

//go:embed ../../carbonio-storages
var CarbonioStoragesDir embed.FS

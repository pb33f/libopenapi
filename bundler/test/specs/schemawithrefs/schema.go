package schemawithrefs

import (
	"embed"
)

//go:embed openapi.yaml
var Schema []byte

//go:embed description.yaml resources
var Files embed.FS

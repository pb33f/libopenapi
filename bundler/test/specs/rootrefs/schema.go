package rootrefs

import "embed"

//go:embed openapi.yaml
var Schema []byte

//go:embed openapi.yaml resources
var Files embed.FS

package v3

import "github.com/pb33f/libopenapi/datamodel/low"

type Document struct {
    Version      string
    Info         Info
    Servers      []Server
    Paths        Paths
    Components   Components
    Security     []SecurityRequirement
    Tags         []Tag
    ExternalDocs ExternalDoc
    Extensions   map[string]low.ObjectReference
}

package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
)

type Document struct {
    Version      low.NodeReference[string]
    Info         low.NodeReference[*Info]
    Servers      []low.NodeReference[*Server]
    Paths        *Paths
    Components   *Components
    Security     []*SecurityRequirement
    Tags         []*Tag
    ExternalDocs *ExternalDoc
    Extensions   map[string]low.ObjectReference
}

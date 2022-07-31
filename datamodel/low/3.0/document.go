package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
)

type Document struct {
	Version      low.NodeReference[string]
	Info         low.NodeReference[*Info]
	Servers      []low.NodeReference[*Server]
	Paths        low.NodeReference[*Paths]
	Components   *Components
	Security     []*SecurityRequirement
	Tags         []low.NodeReference[*Tag]
	ExternalDocs *ExternalDoc
	Extensions   map[low.NodeReference[string]]low.NodeReference[any]
}

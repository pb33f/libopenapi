package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
)

type ExternalDoc struct {
	Description low.NodeReference[string]
	URL         low.NodeReference[string]
	Extensions  map[string]low.ObjectReference
}

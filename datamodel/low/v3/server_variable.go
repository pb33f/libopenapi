package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
)

type ServerVariable struct {
	Enum        []low.NodeReference[string]
	Default     low.NodeReference[string]
	Description low.NodeReference[string]
}

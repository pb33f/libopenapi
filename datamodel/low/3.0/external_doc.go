package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type ExternalDoc struct {
    Node        *yaml.Node
    Description low.NodeReference[string]
    URL         low.NodeReference[string]
    Extensions  map[string]low.ObjectReference
}

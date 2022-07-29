package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Tag struct {
    Node         *yaml.Node
    Name         low.NodeReference[string]
    Description  low.NodeReference[string]
    ExternalDocs ExternalDoc
    Extensions   map[string]low.ObjectReference
}

package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Example struct {
    Node          *yaml.Node
    Summary       low.NodeReference[string]
    Description   low.NodeReference[string]
    Value         low.ObjectReference
    ExternalValue low.NodeReference[string]
    Extensions    map[string]low.ObjectReference
}

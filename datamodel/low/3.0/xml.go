package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type XML struct {
    Node       *yaml.Node
    Name       low.NodeReference[string]
    Namespace  low.NodeReference[string]
    Prefix     low.NodeReference[string]
    Attribute  low.NodeReference[string]
    Wrapped    low.NodeReference[bool]
    Extensions map[string]low.ObjectReference
}

package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type ServerVariable struct {
    Node        *yaml.Node
    Enum        []low.NodeReference[string]
    Default     low.NodeReference[string]
    Description low.NodeReference[string]
}

package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Contact struct {
    Node  *yaml.Node
    Name  low.NodeReference[string]
    URL   low.NodeReference[string]
    Email low.NodeReference[string]
}

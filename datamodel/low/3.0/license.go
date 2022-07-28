package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type License struct {
    Node *yaml.Node
    Name low.NodeReference[string]
    URL  low.NodeReference[string]
}

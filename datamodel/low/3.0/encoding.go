package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Encoding struct {
    Node          *yaml.Node
    ContentType   low.NodeReference[string]
    Headers       map[string]Parameter
    Style         low.NodeReference[string]
    Explode       low.NodeReference[bool]
    AllowReserved low.NodeReference[bool]
}

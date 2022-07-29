package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Discriminator struct {
    Node         *yaml.Node
    PropertyName low.NodeReference[string]
    Mapping      map[string]string
}

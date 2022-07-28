package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Parameter struct {
    Node            *yaml.Node
    Name            low.NodeReference[string]
    In              low.NodeReference[string]
    Description     low.NodeReference[string]
    Required        low.NodeReference[bool]
    Deprecated      low.NodeReference[bool]
    AllowEmptyValue low.NodeReference[bool]
}

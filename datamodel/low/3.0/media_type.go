package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type MediaType struct {
    Node     *yaml.Node
    Schema   Schema
    Example  low.ObjectReference
    Examples map[string]Example
    Encoding map[string]Encoding
}

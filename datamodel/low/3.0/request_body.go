package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type RequestBody struct {
    Node        *yaml.Node
    Description low.NodeReference[string]
    Content     map[string]MediaType
    Required    low.NodeReference[bool]
    Extensions  map[string]low.ObjectReference
}

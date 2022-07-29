package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Link struct {
    Node         *yaml.Node
    OperationRef low.NodeReference[string]
    OperationId  low.NodeReference[string]
    Parameters   map[string]low.NodeReference[string]
    RequestBody  low.NodeReference[string]
    Description  low.NodeReference[string]
    Server       Server
    Extensions   map[string]low.ObjectReference
}

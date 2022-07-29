package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Operation struct {
    Node         *yaml.Node
    Tags         []low.NodeReference[string]
    Summary      low.NodeReference[string]
    Description  low.NodeReference[string]
    ExternalDocs ExternalDoc
    OperationId  low.NodeReference[string]
    Parameters   []Parameter
    RequestBody  RequestBody
    Responses    Responses
    Callbacks    map[string]Callback
    Deprecated   low.NodeReference[bool]
    Security     []SecurityRequirement
    Servers      []Server
    Extensions   map[string]low.ObjectReference
}

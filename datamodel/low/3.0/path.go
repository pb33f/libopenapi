package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Paths struct {
    Node       *yaml.Node
    Paths      map[string]Path
    Extensions map[string]low.ObjectReference
}

type Path struct {
    Node        *yaml.Node
    Value       low.NodeReference[string]
    Summary     low.NodeReference[string]
    Description low.NodeReference[string]
    Get         Operation
    Put         Operation
    Post        Operation
    Delete      Operation
    Options     Operation
    Head        Operation
    Patch       Operation
    Trace       Operation
    Servers     []Server
    Parameters  []Parameter
    Extensions  map[string]low.ObjectReference
}

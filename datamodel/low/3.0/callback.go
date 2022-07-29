package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Callback struct {
    Node       *yaml.Node
    Expression map[string]Path
    Extensions map[string]low.ObjectReference
}

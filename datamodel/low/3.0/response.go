package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Response struct {
    Node        *yaml.Node
    Description low.NodeReference[string]
    Headers     map[string]Parameter
    Content     map[string]MediaType
}

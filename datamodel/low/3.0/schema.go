package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Schema struct {
    Node                 *yaml.Node
    Title                low.NodeReference[string]
    MultipleOf           low.NodeReference[int]
    Maximum              low.NodeReference[int]
    ExclusiveMaximum     low.NodeReference[int]
    Minimum              low.NodeReference[int]
    ExclusiveMinimum     low.NodeReference[int]
    MaxLength            low.NodeReference[int]
    MinLength            low.NodeReference[int]
    Pattern              low.NodeReference[string]
    MaxItems             low.NodeReference[int]
    MinItems             low.NodeReference[int]
    UniqueItems          low.NodeReference[int]
    MaxProperties        low.NodeReference[int]
    MinProperties        low.NodeReference[int]
    Required             []low.NodeReference[string]
    Enum                 []low.NodeReference[string]
    Type                 low.NodeReference[string]
    AllOf                *Schema
    OneOf                *Schema
    AnyOf                *Schema
    Not                  *Schema
    Items                *Schema
    Properties           map[string]*Schema
    AdditionalProperties low.ObjectReference
    Description          low.NodeReference[string]
    Default              low.ObjectReference
}

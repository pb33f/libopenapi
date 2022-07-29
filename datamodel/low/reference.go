package low

import "gopkg.in/yaml.v3"

type HasNode interface {
    GetNode() *yaml.Node
}

type Buildable interface {
    Build(node *yaml.Node)
}

type NodeReference[T comparable] struct {
    Value T
    Node  *yaml.Node
}

type ObjectReference struct {
    Value interface{}
    Node  *yaml.Node
}

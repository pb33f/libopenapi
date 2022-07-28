package low

import "gopkg.in/yaml.v3"

type HasNode interface {
    GetNode() *yaml.Node
}

type NodeReference[T comparable] interface {
    GetValue() T
    GetNode() *yaml.Node
}

type ObjectReference interface {
    GetValue() interface{}
    GetNode() *yaml.Node
}

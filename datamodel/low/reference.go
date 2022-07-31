package low

import "gopkg.in/yaml.v3"

type HasNode interface {
	GetNode() *yaml.Node
}

type Buildable interface {
	Build(node *yaml.Node) error
}

type NodeReference[T any] struct {
	Value     T
	ValueNode *yaml.Node
	KeyNode   *yaml.Node
}

type ObjectReference struct {
	Value     interface{}
	ValueNode *yaml.Node
	KeyNode   *yaml.Node
}

package low

import "gopkg.in/yaml.v3"

type HasNode interface {
	GetNode() *yaml.Node
}

type Buildable[T any] interface {
	Build(node *yaml.Node) error
	*T
}

type NodeReference[T any] struct {
	Value     T
	ValueNode *yaml.Node
	KeyNode   *yaml.Node
}

type KeyReference[T any] struct {
	Value   T
	KeyNode *yaml.Node
}

type ValueReference[T any] struct {
	Value     T
	ValueNode *yaml.Node
}

type ObjectReference struct {
	Value     interface{}
	ValueNode *yaml.Node
	KeyNode   *yaml.Node
}

func (n NodeReference[T]) IsEmpty() bool {
	return n.KeyNode == nil && n.ValueNode == nil
}

func (n NodeReference[T]) IsMapKeyNode() bool {
	return n.KeyNode != nil && n.ValueNode == nil
}

func (n NodeReference[T]) IsMapValueNode() bool {
	return n.KeyNode == nil && n.ValueNode != nil
}

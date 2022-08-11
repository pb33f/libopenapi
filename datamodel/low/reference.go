package low

import (
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

type HasNode interface {
	GetNode() *yaml.Node
}

type Buildable[T any] interface {
	Build(node *yaml.Node, idx *index.SpecIndex) error
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

func (n NodeReference[T]) IsEmpty() bool {
	return n.KeyNode == nil && n.ValueNode == nil
}

func (n ValueReference[T]) IsEmpty() bool {
	return n.ValueNode == nil
}

func (n KeyReference[T]) IsEmpty() bool {
	return n.KeyNode == nil
}

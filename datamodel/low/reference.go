package low

import (
	"fmt"
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

func (n NodeReference[T]) GenerateMapKey() string {
	return fmt.Sprintf("%d:%d", n.ValueNode.Line, n.ValueNode.Column)
}

func (n ValueReference[T]) IsEmpty() bool {
	return n.ValueNode == nil
}

func (n ValueReference[T]) GenerateMapKey() string {
	return fmt.Sprintf("%d:%d", n.ValueNode.Line, n.ValueNode.Column)
}

func (n KeyReference[T]) IsEmpty() bool {
	return n.KeyNode == nil
}

func (n KeyReference[T]) GenerateMapKey() string {
	return fmt.Sprintf("%d:%d", n.KeyNode.Line, n.KeyNode.Column)
}

func IsCircular(node *yaml.Node, idx *index.SpecIndex) bool {
	if idx == nil {
		return false // no index! nothing we can do.
	}
	refs := idx.GetCircularReferences()
	for i := range idx.GetCircularReferences() {
		if refs[i].LoopPoint.Node == node {
			return true
		}
	}
	return false
}

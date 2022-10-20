package low

import (
	"fmt"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// Buildable is an interface for any struct that can be 'built out'. This means that a struct can accept
// a root node and a reference to the index that carries data about any references used.
//
// Used by generic functions when automatically building out structs based on yaml.Node inputs.
type Buildable[T any] interface {
	Build(node *yaml.Node, idx *index.SpecIndex) error
	*T
}

// HasValueNode is implemented by NodeReference and ValueReference to return the yaml.Node backing the value.
type HasValueNode[T any] interface {
	GetValueNode() *yaml.Node
	*T
}

// Hashable defines any struct that implements a Hash function that returns a 256SHA hash of the state of the
// representative object. Great for equality checking!
type Hashable interface {
	Hash() [32]byte
}

// HasExtensions is implemented by any object that exposes extensions
type HasExtensions[T any] interface {
	GetExtensions() map[KeyReference[string]]ValueReference[any]
}

// HasValue is implemented by NodeReference and ValueReference to return the yaml.Node backing the value.
type HasValue[T any] interface {
	GetValue() T
	GetValueNode() *yaml.Node
	*T
}

// NodeReference is a low-level container for holding a Value of type T, as well as references to
// a key yaml.Node that points to the key node that contains the value node, and the value node that contains
// the actual value.
type NodeReference[T any] struct {

	// The value being referenced
	Value T

	// The yaml.Node that holds the value
	ValueNode *yaml.Node

	// The yaml.Node that is the key, that contains the value.
	KeyNode *yaml.Node
}

// KeyReference is a low-level container for key nodes holding a Value of type T. A KeyNode is a pointer to the
// yaml.Node that holds a key to a value.
type KeyReference[T any] struct {

	// The value being referenced.
	Value T

	// The yaml.Node that holds this referenced key
	KeyNode *yaml.Node
}

// ValueReference is a low-level container for value nodes that hold a Value of type T. A ValueNode is a pointer
// to the yaml.Node that holds the value.
type ValueReference[T any] struct {

	// The value being referenced.
	Value T

	// The yaml.Node that holds the referenced value
	ValueNode *yaml.Node
}

// IsEmpty will return true if this reference has no key or value nodes assigned (it's been ignored)
func (n NodeReference[T]) IsEmpty() bool {
	return n.KeyNode == nil && n.ValueNode == nil
}

func (n NodeReference[T]) IsReferenceNode() bool {
	for k := range n.KeyNode.Content {
		if k%2 == 0 {
			if n.KeyNode.Content[k].Value == "$ref" {
				return true
			}
		}
	}
	return false
}

// GenerateMapKey will return a string based on the line and column number of the node, e.g. 33:56 for line 33, col 56.
func (n NodeReference[T]) GenerateMapKey() string {
	return fmt.Sprintf("%d:%d", n.ValueNode.Line, n.ValueNode.Column)
}

// Mutate will set the reference value to what is supplied. This happens to both the Value and ValueNode, which means
// the root document is permanently mutated and changes will be reflected in any serialization of the root document.
func (n NodeReference[T]) Mutate(value T) NodeReference[T] {
	n.ValueNode.Value = fmt.Sprintf("%v", value)
	n.Value = value
	return n
}

// GetValueNode will return the yaml.Node containing the reference value node
func (n NodeReference[T]) GetValueNode() *yaml.Node {
	return n.ValueNode
}

// GetValue will return the  raw value of the node
func (n NodeReference[T]) GetValue() T {
	return n.Value
}

// IsEmpty will return true if this reference has no key or value nodes assigned (it's been ignored)
func (n ValueReference[T]) IsEmpty() bool {
	return n.ValueNode == nil
}

// GenerateMapKey will return a string based on the line and column number of the node, e.g. 33:56 for line 33, col 56.
func (n ValueReference[T]) GenerateMapKey() string {
	return fmt.Sprintf("%d:%d", n.ValueNode.Line, n.ValueNode.Column)
}

// GetValueNode will return the yaml.Node containing the reference value node
func (n ValueReference[T]) GetValueNode() *yaml.Node {
	return n.ValueNode
}

// GetValue will return the  raw value of the node
func (n ValueReference[T]) GetValue() T {
	return n.Value
}

// IsEmpty will return true if this reference has no key or value nodes assigned (it's been ignored)
func (n KeyReference[T]) IsEmpty() bool {
	return n.KeyNode == nil
}

// GenerateMapKey will return a string based on the line and column number of the node, e.g. 33:56 for line 33, col 56.
func (n KeyReference[T]) GenerateMapKey() string {
	return fmt.Sprintf("%d:%d", n.KeyNode.Line, n.KeyNode.Column)
}

// Mutate will set the reference value to what is supplied. This happens to both the Value and ValueNode, which means
// the root document is permanently mutated and changes will be reflected in any serialization of the root document.
func (n ValueReference[T]) Mutate(value T) ValueReference[T] {
	n.ValueNode.Value = fmt.Sprintf("%v", value)
	n.Value = value
	return n
}

// IsCircular will determine if the node in question, is part of a circular reference chain discovered by the index.
func IsCircular(node *yaml.Node, idx *index.SpecIndex) bool {
	if idx == nil {
		return false // no index! nothing we can do.
	}
	refs := idx.GetCircularReferences()
	for i := range idx.GetCircularReferences() {
		if refs[i].LoopPoint.Node == node {
			return true
		}
		for k := range refs[i].Journey {
			if refs[i].Journey[k].Node == node {
				return true
			}
		}
	}
	// check mapped references in case we didn't find it.
	_, nv := utils.FindKeyNode("$ref", node.Content)
	if nv != nil {
		ref := idx.GetMappedReferences()[nv.Value]
		if ref != nil {
			return ref.Circular
		}
	}
	return false
}

// GetCircularReferenceResult will check if a node is part of a circular reference chain and then return that
// index.CircularReferenceResult it was located in. Returns nil if not found.
func GetCircularReferenceResult(node *yaml.Node, idx *index.SpecIndex) *index.CircularReferenceResult {
	if idx == nil {
		return nil // no index! nothing we can do.
	}
	refs := idx.GetCircularReferences()
	for i := range refs {
		if refs[i].LoopPoint.Node == node {
			return refs[i]
		}
		for k := range refs[i].Journey {
			if refs[i].Journey[k].Node == node {
				return refs[i]
			}
		}
	}
	// check mapped references in case we didn't find it.
	_, nv := utils.FindKeyNode("$ref", node.Content)
	if nv != nil {
		for i := range refs {
			if refs[i].LoopPoint.Definition == nv.Value {
				return refs[i]
			}
		}
	}
	return nil
}

func HashToString(hash [32]byte) string {
	return fmt.Sprintf("%x", hash)
}

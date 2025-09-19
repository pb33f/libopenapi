package base

import (
	"context"
	"crypto/sha256"
	"strconv"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

// XML represents a low-level representation of an XML object defined by all versions of OpenAPI.
//
// A metadata object that allows for more fine-tuned XML model definitions.
//
// When using arrays, XML element names are not inferred (for singular/plural forms) and the name property SHOULD be
// used to add that information. See examples for expected behavior.
//
//	v2 - https://swagger.io/specification/v2/#xmlObject
//	v3 - https://swagger.io/specification/#xml-object
type XML struct {
	Name       low.NodeReference[string]
	Namespace  low.NodeReference[string]
	Prefix     low.NodeReference[string]
	Attribute  low.NodeReference[bool]
	NodeType   low.NodeReference[string] // OpenAPI 3.2+ nodeType field (replaces deprecated attribute field)
	Wrapped    low.NodeReference[bool]
	Extensions *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	RootNode   *yaml.Node
	index      *index.SpecIndex
	context    context.Context
	*low.Reference
	low.NodeMap
}

// Build will extract extensions from the XML instance.
func (x *XML) Build(root *yaml.Node, idx *index.SpecIndex) error {
	root = utils.NodeAlias(root)
	utils.CheckForMergeNodes(root)
	x.RootNode = root
	x.Reference = new(low.Reference)
	x.Nodes = low.ExtractNodes(nil, root)
	x.Extensions = low.ExtractExtensions(root)
	x.index = idx
	return nil
}

// GetExtensions returns all Tag extensions and satisfies the low.HasExtensions interface.
func (x *XML) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return x.Extensions
}

// GetRootNode returns the root yaml node of the Tag object
func (x *XML) GetRootNode() *yaml.Node {
	return x.RootNode
}

// GetIndex returns the index of the XML object
func (x *XML) GetIndex() *index.SpecIndex {
	return x.index
}

// Hash generates a SHA256 hash of the XML object using properties
func (x *XML) Hash() [32]byte {
	// Use string builder pool
	sb := low.GetStringBuilder()
	defer low.PutStringBuilder(sb)

	if !x.Name.IsEmpty() {
		sb.WriteString(x.Name.Value)
		sb.WriteByte('|')
	}
	if !x.Namespace.IsEmpty() {
		sb.WriteString(x.Namespace.Value)
		sb.WriteByte('|')
	}
	if !x.Prefix.IsEmpty() {
		sb.WriteString(x.Prefix.Value)
		sb.WriteByte('|')
	}
	if !x.Attribute.IsEmpty() {
		sb.WriteString(strconv.FormatBool(x.Attribute.Value))
		sb.WriteByte('|')
	}
	if !x.NodeType.IsEmpty() {
		sb.WriteString(x.NodeType.Value)
		sb.WriteByte('|')
	}
	if !x.Wrapped.IsEmpty() {
		sb.WriteString(strconv.FormatBool(x.Wrapped.Value))
		sb.WriteByte('|')
	}
	for _, ext := range low.HashExtensions(x.Extensions) {
		sb.WriteString(ext)
		sb.WriteByte('|')
	}
	return sha256.Sum256([]byte(sb.String()))
}

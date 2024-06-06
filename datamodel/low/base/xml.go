package base

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
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
	Wrapped    low.NodeReference[bool]
	Extensions *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	RootNode   *yaml.Node
	*low.Reference
}

// Build will extract extensions from the XML instance.
func (x *XML) Build(root *yaml.Node, _ *index.SpecIndex) error {
	root = utils.NodeAlias(root)
	utils.CheckForMergeNodes(root)
	x.RootNode = root
	x.Reference = new(low.Reference)
	x.Extensions = low.ExtractExtensions(root)
	return nil
}

// GetExtensions returns all Tag extensions and satisfies the low.HasExtensions interface.
func (x *XML) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return x.Extensions
}

// Hash generates a SHA256 hash of the XML object using properties
func (x *XML) Hash() [32]byte {
	var f []string
	if !x.Name.IsEmpty() {
		f = append(f, x.Name.Value)
	}
	if !x.Namespace.IsEmpty() {
		f = append(f, x.Namespace.Value)
	}
	if !x.Prefix.IsEmpty() {
		f = append(f, x.Prefix.Value)
	}
	if !x.Attribute.IsEmpty() {
		f = append(f, fmt.Sprint(x.Attribute.Value))
	}
	if !x.Wrapped.IsEmpty() {
		f = append(f, fmt.Sprint(x.Wrapped.Value))
	}
	f = append(f, low.HashExtensions(x.Extensions)...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

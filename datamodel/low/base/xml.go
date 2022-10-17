package base

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"strings"
)

// XML represents a low-level representation of an XML object defined by all versions of OpenAPI.
//
// A metadata object that allows for more fine-tuned XML model definitions.
//
// When using arrays, XML element names are not inferred (for singular/plural forms) and the name property SHOULD be
// used to add that information. See examples for expected behavior.
//  v2 - https://swagger.io/specification/v2/#xmlObject
//  v3 - https://swagger.io/specification/#xml-object
type XML struct {
	Name       low.NodeReference[string]
	Namespace  low.NodeReference[string]
	Prefix     low.NodeReference[string]
	Attribute  low.NodeReference[bool]
	Wrapped    low.NodeReference[bool]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

// Build will extract extensions from the XML instance.
func (x *XML) Build(root *yaml.Node, _ *index.SpecIndex) error {
	x.Extensions = low.ExtractExtensions(root)
	return nil
}

func (x *XML) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return x.Extensions
}

// Hash generates a SHA256 hash of the XML object using properties
func (x *XML) Hash() [32]byte {
	// calculate a hash from every property.
	d := []string{
		x.Name.Value,
		x.Namespace.Value,
		x.Prefix.Value,
		fmt.Sprint(x.Attribute.Value),
		fmt.Sprint(x.Wrapped.Value),
	}
	// add extensions to hash
	for k := range x.Extensions {
		d = append(d, fmt.Sprintf("%v-%x", k.Value, x.Extensions[k].Value))
	}
	return sha256.Sum256([]byte(strings.Join(d, "|")))
}

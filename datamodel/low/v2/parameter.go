// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// Parameter represents a low-level Swagger / OpenAPI 2 Parameter object.
//
// A unique parameter is defined by a combination of a name and location.
//
// There are five possible parameter types.
//
// Path
//   Used together with Path Templating, where the parameter value is actually part of the operation's URL.
//   This does not include the host or base path of the API. For example, in /items/{itemId}, the path parameter is itemId.
// Query
//   Parameters that are appended to the URL. For example, in /items?id=###, the query parameter is id.
// Header
//   Custom headers that are expected as part of the request.
// Body
//   The payload that's appended to the HTTP request. Since there can only be one payload, there can only be one body parameter.
//   The name of the body parameter has no effect on the parameter itself and is used for documentation purposes only.
//   Since Form parameters are also in the payload, body and form parameters cannot exist together for the same operation.
// Form
//   Used to describe the payload of an HTTP request when either application/x-www-form-urlencoded, multipart/form-data
//   or both are used as the content type of the request (in Swagger's definition, the consumes property of an operation).
//   This is the only parameter type that can be used to send files, thus supporting the file type. Since form parameters
//   are sent in the payload, they cannot be declared together with a body parameter for the same operation. Form
//   parameters have a different format based on the content-type used (for further details,
//   consult http://www.w3.org/TR/html401/interact/forms.html#h-17.13.4):
//     application/x-www-form-urlencoded - Similar to the format of Query parameters but as a payload. For example,
//     foo=1&bar=swagger - both foo and bar are form parameters. This is normally used for simple parameters that are
//                         being transferred.
//     multipart/form-data - each parameter takes a section in the payload with an internal header. For example, for
//                           the header Content-Disposition: form-data; name="submit-name" the name of the parameter is
//                           submit-name. This type of form parameters is more commonly used for file transfers
// https://swagger.io/specification/v2/#parameterObject
type Parameter struct {
	Name             low.NodeReference[string]
	In               low.NodeReference[string]
	Type             low.NodeReference[string]
	Format           low.NodeReference[string]
	Description      low.NodeReference[string]
	Required         low.NodeReference[bool]
	AllowEmptyValue  low.NodeReference[bool]
	Schema           low.NodeReference[*base.SchemaProxy]
	Items            low.NodeReference[*Items]
	CollectionFormat low.NodeReference[string]
	Default          low.NodeReference[any]
	Maximum          low.NodeReference[int]
	ExclusiveMaximum low.NodeReference[bool]
	Minimum          low.NodeReference[int]
	ExclusiveMinimum low.NodeReference[bool]
	MaxLength        low.NodeReference[int]
	MinLength        low.NodeReference[int]
	Pattern          low.NodeReference[string]
	MaxItems         low.NodeReference[int]
	MinItems         low.NodeReference[int]
	UniqueItems      low.NodeReference[bool]
	Enum             low.NodeReference[[]low.ValueReference[string]]
	MultipleOf       low.NodeReference[int]
	Extensions       map[low.KeyReference[string]]low.ValueReference[any]
}

// FindExtension attempts to locate a extension value given a name.
func (p *Parameter) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, p.Extensions)
}

// Build will extract out extensions, schema, items and default value
func (p *Parameter) Build(root *yaml.Node, idx *index.SpecIndex) error {
	p.Extensions = low.ExtractExtensions(root)
	sch, sErr := base.ExtractSchema(root, idx)
	if sErr != nil {
		return sErr
	}
	if sch != nil {
		p.Schema = *sch
	}
	items, iErr := low.ExtractObject[*Items](ItemsLabel, root, idx)
	if iErr != nil {
		return iErr
	}
	p.Items = items

	_, ln, vn := utils.FindKeyNodeFull(DefaultLabel, root.Content)
	if vn != nil {
		var n map[string]interface{}
		err := vn.Decode(&n)
		if err != nil {
			var k []interface{}
			err = vn.Decode(&k)
			if err != nil {
				var j interface{}
				_ = vn.Decode(&j)
				p.Default = low.NodeReference[any]{
					Value:     j,
					KeyNode:   ln,
					ValueNode: vn,
				}
				return nil
			}
			p.Default = low.NodeReference[any]{
				Value:     k,
				KeyNode:   ln,
				ValueNode: vn,
			}
			return nil
		}
		p.Default = low.NodeReference[any]{
			Value:     n,
			KeyNode:   ln,
			ValueNode: vn,
		}
		return nil
	}
	return nil
}

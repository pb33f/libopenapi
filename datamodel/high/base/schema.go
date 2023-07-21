// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"gopkg.in/yaml.v3"
)

// Schema represents a JSON Schema that support Swagger, OpenAPI 3 and OpenAPI 3.1
//
// Until 3.1 OpenAPI had a strange relationship with JSON Schema. It's been a super-set/sub-set
// mix, which has been confusing. So, instead of building a bunch of different models, we have compressed
// all variations into a single model that makes it easy to support multiple spec types.
//
//   - v2 schema: https://swagger.io/specification/v2/#schemaObject
//   - v3 schema: https://swagger.io/specification/#schema-object
//   - v3.1 schema: https://spec.openapis.org/oas/v3.1.0#schema-object
type Schema struct {
	// 3.1 only, used to define a dialect for this schema, label is '$schema'.
	SchemaTypeRef string `json:"$schema,omitempty" yaml:"$schema,omitempty"`

	// In versions 2 and 3.0, this ExclusiveMaximum can only be a boolean.
	// In version 3.1, ExclusiveMaximum is a number.
	ExclusiveMaximum *DynamicValue[bool, float64] `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`

	// In versions 2 and 3.0, this ExclusiveMinimum can only be a boolean.
	// In version 3.1, ExclusiveMinimum is a number.
	ExclusiveMinimum *DynamicValue[bool, float64] `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`

	// In versions 2 and 3.0, this Type is a single value, so array will only ever have one value
	// in version 3.1, Type can be multiple values
	Type []string `json:"type,omitempty" yaml:"type,omitempty"`

	// Schemas are resolved on demand using a SchemaProxy
	AllOf []*SchemaProxy `json:"allOf,omitempty" yaml:"allOf,omitempty"`

	// Polymorphic Schemas are only available in version 3+
	OneOf         []*SchemaProxy `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf         []*SchemaProxy `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	Discriminator *Discriminator `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`

	// in 3.1 examples can be an array (which is recommended)
	Examples []any `json:"examples,omitempty" yaml:"examples,omitempty"`

	// in 3.1 prefixItems provides tuple validation support.
	PrefixItems []*SchemaProxy `json:"prefixItems,omitempty" yaml:"prefixItems,omitempty"`

	// 3.1 Specific properties
	Contains          *SchemaProxy            `json:"contains,omitempty" yaml:"contains,omitempty"`
	MinContains       *int64                  `json:"minContains,omitempty" yaml:"minContains,omitempty"`
	MaxContains       *int64                  `json:"maxContains,omitempty" yaml:"maxContains,omitempty"`
	If                *SchemaProxy            `json:"if,omitempty" yaml:"if,omitempty"`
	Else              *SchemaProxy            `json:"else,omitempty" yaml:"else,omitempty"`
	Then              *SchemaProxy            `json:"then,omitempty" yaml:"then,omitempty"`
	DependentSchemas  map[string]*SchemaProxy `json:"dependentSchemas,omitempty" yaml:"dependentSchemas,omitempty"`
	PatternProperties map[string]*SchemaProxy `json:"patternProperties,omitempty" yaml:"patternProperties,omitempty"`
	PropertyNames     *SchemaProxy            `json:"propertyNames,omitempty" yaml:"propertyNames,omitempty"`
	UnevaluatedItems  *SchemaProxy            `json:"unevaluatedItems,omitempty" yaml:"unevaluatedItems,omitempty"`

	// in 3.1 UnevaluatedProperties can be a Schema or a boolean
	// https://github.com/pb33f/libopenapi/issues/118
	UnevaluatedProperties *DynamicValue[*SchemaProxy, *bool] `json:"unevaluatedProperties,omitempty" yaml:"unevaluatedProperties,omitempty"`

	// in 3.1 Items can be a Schema or a boolean
	Items *DynamicValue[*SchemaProxy, bool] `json:"items,omitempty" yaml:"items,omitempty"`

	// 3.1 only, part of the JSON Schema spec provides a way to identify a subschema
	Anchor string `json:"$anchor,omitempty" yaml:"$anchor,omitempty"`

	// Compatible with all versions
	Not                  *SchemaProxy            `json:"not,omitempty" yaml:"not,omitempty"`
	Properties           map[string]*SchemaProxy `json:"properties,omitempty" yaml:"properties,omitempty"`
	Title                string                  `json:"title,omitempty" yaml:"title,omitempty"`
	MultipleOf           *float64                `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Maximum              *float64                `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	Minimum              *float64                `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	MaxLength            *int64                  `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinLength            *int64                  `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	Pattern              string                  `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Format               string                  `json:"format,omitempty" yaml:"format,omitempty"`
	MaxItems             *int64                  `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MinItems             *int64                  `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	UniqueItems          *bool                   `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	MaxProperties        *int64                  `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	MinProperties        *int64                  `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	Required             []string                `json:"required,omitempty" yaml:"required,omitempty"`
	Enum                 []any                   `json:"enum,omitempty" yaml:"enum,omitempty"`
	AdditionalProperties any                     `json:"additionalProperties,omitempty" yaml:"additionalProperties,renderZero,omitempty"`
	Description          string                  `json:"description,omitempty" yaml:"description,omitempty"`
	Default              any                     `json:"default,omitempty" yaml:"default,renderZero,omitempty"`
	Nullable             *bool                   `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	ReadOnly             bool                    `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`   // https://github.com/pb33f/libopenapi/issues/30
	WriteOnly            bool                    `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"` // https://github.com/pb33f/libopenapi/issues/30
	XML                  *XML                    `json:"xml,omitempty" yaml:"xml,omitempty"`
	ExternalDocs         *ExternalDoc            `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	Example              any                     `json:"example,omitempty" yaml:"example,omitempty"`
	Deprecated           *bool                   `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Extensions           map[string]any          `json:"-" yaml:"-"`
	low                  *base.Schema

	// Parent Proxy refers back to the low level SchemaProxy that is proxying this schema.
	ParentProxy *SchemaProxy `json:"-" yaml:"-"`
}

// NewSchema will create a new high-level schema from a low-level one.
func NewSchema(schema *base.Schema) *Schema {
	s := new(Schema)
	s.low = schema
	s.Title = schema.Title.Value
	if !schema.MultipleOf.IsEmpty() {
		s.MultipleOf = &schema.MultipleOf.Value
	}
	if !schema.Maximum.IsEmpty() {
		s.Maximum = &schema.Maximum.Value
	}
	if !schema.Minimum.IsEmpty() {
		s.Minimum = &schema.Minimum.Value
	}
	// if we're dealing with a 3.0 spec using a bool
	if !schema.ExclusiveMaximum.IsEmpty() && schema.ExclusiveMaximum.Value.IsA() {
		s.ExclusiveMaximum = &DynamicValue[bool, float64]{
			A: schema.ExclusiveMaximum.Value.A,
		}
	}
	// if we're dealing with a 3.1 spec using an int
	if !schema.ExclusiveMaximum.IsEmpty() && schema.ExclusiveMaximum.Value.IsB() {
		s.ExclusiveMaximum = &DynamicValue[bool, float64]{
			N: 1,
			B: schema.ExclusiveMaximum.Value.B,
		}
	}
	// if we're dealing with a 3.0 spec using a bool
	if !schema.ExclusiveMinimum.IsEmpty() && schema.ExclusiveMinimum.Value.IsA() {
		s.ExclusiveMinimum = &DynamicValue[bool, float64]{
			A: schema.ExclusiveMinimum.Value.A,
		}
	}
	// if we're dealing with a 3.1 spec, using an int
	if !schema.ExclusiveMinimum.IsEmpty() && schema.ExclusiveMinimum.Value.IsB() {
		s.ExclusiveMinimum = &DynamicValue[bool, float64]{
			N: 1,
			B: schema.ExclusiveMinimum.Value.B,
		}
	}
	if !schema.MaxLength.IsEmpty() {
		s.MaxLength = &schema.MaxLength.Value
	}
	if !schema.MinLength.IsEmpty() {
		s.MinLength = &schema.MinLength.Value
	}
	if !schema.MaxItems.IsEmpty() {
		s.MaxItems = &schema.MaxItems.Value
	}
	if !schema.MinItems.IsEmpty() {
		s.MinItems = &schema.MinItems.Value
	}
	if !schema.MaxProperties.IsEmpty() {
		s.MaxProperties = &schema.MaxProperties.Value
	}
	if !schema.MinProperties.IsEmpty() {
		s.MinProperties = &schema.MinProperties.Value
	}

	if !schema.MaxContains.IsEmpty() {
		s.MaxContains = &schema.MaxContains.Value
	}
	if !schema.MinContains.IsEmpty() {
		s.MinContains = &schema.MinContains.Value
	}
	if !schema.UniqueItems.IsEmpty() {
		s.UniqueItems = &schema.UniqueItems.Value
	}
	if !schema.Contains.IsEmpty() {
		s.Contains = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.Contains.ValueNode,
			Value:     schema.Contains.Value,
		}}
	}
	if !schema.If.IsEmpty() {
		s.If = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.If.ValueNode,
			Value:     schema.If.Value,
		}}
	}
	if !schema.Else.IsEmpty() {
		s.Else = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.Else.ValueNode,
			Value:     schema.Else.Value,
		}}
	}
	if !schema.Then.IsEmpty() {
		s.Then = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.Then.ValueNode,
			Value:     schema.Then.Value,
		}}
	}
	if !schema.PropertyNames.IsEmpty() {
		s.PropertyNames = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.PropertyNames.ValueNode,
			Value:     schema.PropertyNames.Value,
		}}
	}
	if !schema.UnevaluatedItems.IsEmpty() {
		s.UnevaluatedItems = &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: schema.UnevaluatedItems.ValueNode,
			Value:     schema.UnevaluatedItems.Value,
		}}
	}
	// check if unevaluated properties is a schema
	if !schema.UnevaluatedProperties.IsEmpty() && schema.UnevaluatedProperties.Value.IsA() {
		s.UnevaluatedProperties = &DynamicValue[*SchemaProxy, *bool]{
			A: &SchemaProxy{
				schema: &lowmodel.NodeReference[*base.SchemaProxy]{
					ValueNode: schema.UnevaluatedProperties.ValueNode,
					Value:     schema.UnevaluatedProperties.Value.A,
				},
			},
			N: 0,
		}
	}

	// check if unevaluated properties is a bool
	if !schema.UnevaluatedProperties.IsEmpty() && schema.UnevaluatedProperties.Value.IsB() {
		s.UnevaluatedProperties = &DynamicValue[*SchemaProxy, *bool]{
			B: schema.UnevaluatedProperties.Value.B,
			N: 1,
		}
	}

	if !schema.UnevaluatedProperties.IsEmpty() {

	}

	s.Pattern = schema.Pattern.Value
	s.Format = schema.Format.Value

	// 3.0 spec is a single value
	if !schema.Type.IsEmpty() && schema.Type.Value.IsA() {
		s.Type = []string{schema.Type.Value.A}
	}
	// 3.1 spec may have multiple values
	if !schema.Type.IsEmpty() && schema.Type.Value.IsB() {
		for i := range schema.Type.Value.B {
			s.Type = append(s.Type, schema.Type.Value.B[i].Value)
		}
	}
	if schema.AdditionalProperties.Value != nil {
		if addPropSchema, ok := schema.AdditionalProperties.Value.(*base.SchemaProxy); ok {
			s.AdditionalProperties = NewSchemaProxy(&lowmodel.NodeReference[*base.SchemaProxy]{
				KeyNode:   schema.AdditionalProperties.KeyNode,
				ValueNode: schema.AdditionalProperties.ValueNode,
				Value:     addPropSchema,
			})
		} else {
			// TODO: check for slice and map types and unpack correctly.

			s.AdditionalProperties = schema.AdditionalProperties.Value
		}
	}
	s.Description = schema.Description.Value
	s.Default = schema.Default.Value
	if !schema.Nullable.IsEmpty() {
		s.Nullable = &schema.Nullable.Value
	}
	if !schema.ReadOnly.IsEmpty() {
		s.ReadOnly = schema.ReadOnly.Value
	}
	if !schema.WriteOnly.IsEmpty() {
		s.WriteOnly = schema.WriteOnly.Value
	}
	if !schema.Deprecated.IsEmpty() {
		s.Deprecated = &schema.Deprecated.Value
	}
	s.Example = schema.Example.Value
	if len(schema.Examples.Value) > 0 {
		examples := make([]any, len(schema.Examples.Value))
		for i := 0; i < len(schema.Examples.Value); i++ {
			examples[i] = schema.Examples.Value[i].Value
		}
		s.Examples = examples
	}
	s.Extensions = high.ExtractExtensions(schema.Extensions)
	if !schema.Discriminator.IsEmpty() {
		s.Discriminator = NewDiscriminator(schema.Discriminator.Value)
	}
	if !schema.XML.IsEmpty() {
		s.XML = NewXML(schema.XML.Value)
	}
	if !schema.ExternalDocs.IsEmpty() {
		s.ExternalDocs = NewExternalDoc(schema.ExternalDocs.Value)
	}
	var req []string
	for i := range schema.Required.Value {
		req = append(req, schema.Required.Value[i].Value)
	}
	s.Required = req

	var enum []any
	if !schema.Anchor.IsEmpty() {
		s.Anchor = schema.Anchor.Value
	}

	// TODO: check this behavior.
	for i := range schema.Enum.Value {
		enum = append(enum, schema.Enum.Value[i].Value)
	}
	s.Enum = enum

	// async work.
	// any polymorphic properties need to be handled in their own threads
	// any properties each need to be processed in their own thread.
	// we go as fast as we can.
	polyCompletedChan := make(chan bool)
	errChan := make(chan error)

	type buildResult struct {
		idx int
		s   *SchemaProxy
	}

	// for every item, build schema async
	buildSchema := func(sch lowmodel.ValueReference[*base.SchemaProxy], idx int, bChan chan buildResult) {
		p := &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
			ValueNode: sch.ValueNode,
			Value:     sch.Value,
			Reference: sch.GetReference(),
		}}

		bChan <- buildResult{idx: idx, s: p}
	}

	// schema async
	buildOutSchemas := func(schemas []lowmodel.ValueReference[*base.SchemaProxy], items *[]*SchemaProxy,
		doneChan chan bool, e chan error,
	) {
		bChan := make(chan buildResult)
		totalSchemas := len(schemas)
		for i := range schemas {
			go buildSchema(schemas[i], i, bChan)
		}
		j := 0
		for j < totalSchemas {
			select {
			case r := <-bChan:
				j++
				(*items)[r.idx] = r.s
			}
		}
		doneChan <- true
	}

	// props async
	buildProps := func(k lowmodel.KeyReference[string], v lowmodel.ValueReference[*base.SchemaProxy],
		props map[string]*SchemaProxy, sw int,
	) {
		props[k.Value] = &SchemaProxy{
			schema: &lowmodel.NodeReference[*base.SchemaProxy]{
				Value:     v.Value,
				KeyNode:   k.KeyNode,
				ValueNode: v.ValueNode,
			},
		}

		switch sw {
		case 0:
			s.Properties = props
		case 1:
			s.DependentSchemas = props
		case 2:
			s.PatternProperties = props
		}
	}

	props := make(map[string]*SchemaProxy)
	for k, v := range schema.Properties.Value {
		buildProps(k, v, props, 0)
	}

	dependents := make(map[string]*SchemaProxy)
	for k, v := range schema.DependentSchemas.Value {
		buildProps(k, v, dependents, 1)
	}
	patternProps := make(map[string]*SchemaProxy)
	for k, v := range schema.PatternProperties.Value {
		buildProps(k, v, patternProps, 2)
	}

	var allOf []*SchemaProxy
	var oneOf []*SchemaProxy
	var anyOf []*SchemaProxy
	var not *SchemaProxy
	var items *DynamicValue[*SchemaProxy, bool]
	var prefixItems []*SchemaProxy

	children := 0
	if !schema.AllOf.IsEmpty() {
		children++
		allOf = make([]*SchemaProxy, len(schema.AllOf.Value))
		go buildOutSchemas(schema.AllOf.Value, &allOf, polyCompletedChan, errChan)
	}
	if !schema.AnyOf.IsEmpty() {
		children++
		anyOf = make([]*SchemaProxy, len(schema.AnyOf.Value))
		go buildOutSchemas(schema.AnyOf.Value, &anyOf, polyCompletedChan, errChan)
	}
	if !schema.OneOf.IsEmpty() {
		children++
		oneOf = make([]*SchemaProxy, len(schema.OneOf.Value))
		go buildOutSchemas(schema.OneOf.Value, &oneOf, polyCompletedChan, errChan)
	}
	if !schema.Not.IsEmpty() {
		not = NewSchemaProxy(&schema.Not)
	}
	if !schema.Items.IsEmpty() {
		if schema.Items.Value.IsA() {
			items = &DynamicValue[*SchemaProxy, bool]{A: &SchemaProxy{schema: &lowmodel.NodeReference[*base.SchemaProxy]{
				ValueNode: schema.Items.ValueNode,
				Value:     schema.Items.Value.A,
				KeyNode:   schema.Items.KeyNode,
			}}}
		} else {
			items = &DynamicValue[*SchemaProxy, bool]{N: 1, B: schema.Items.Value.B}
		}
	}
	if !schema.PrefixItems.IsEmpty() {
		children++
		prefixItems = make([]*SchemaProxy, len(schema.PrefixItems.Value))
		go buildOutSchemas(schema.PrefixItems.Value, &prefixItems, polyCompletedChan, errChan)
	}

	completeChildren := 0
	if children > 0 {
	allDone:
		for true {
			select {
			case <-polyCompletedChan:
				completeChildren++
				if children == completeChildren {
					break allDone
				}
			}
		}
	}
	s.OneOf = oneOf
	s.AnyOf = anyOf
	s.AllOf = allOf
	s.Items = items
	s.PrefixItems = prefixItems
	s.Not = not
	return s
}

// GoLow will return the low-level instance of Schema that was used to create the high level one.
func (s *Schema) GoLow() *base.Schema {
	return s.low
}

// GoLowUntyped will return the low-level Schema instance that was used to create the high-level one, with no type
func (s *Schema) GoLowUntyped() any {
	return s.low
}

// Render will return a YAML representation of the Schema object as a byte slice.
func (s *Schema) Render() ([]byte, error) {
	return yaml.Marshal(s)
}

// RenderInline will return a YAML representation of the Schema object as a byte slice. All of the
// $ref values will be inlined, as in resolved in place.
//
// Make sure you don't have any circular references!
func (s *Schema) RenderInline() ([]byte, error) {
	d, _ := s.MarshalYAMLInline()
	return yaml.Marshal(d)
}

// MarshalYAML will create a ready to render YAML representation of the ExternalDoc object.
func (s *Schema) MarshalYAML() (interface{}, error) {
	nb := high.NewNodeBuilder(s, s.low)
	return nb.Render(), nil
}

func (s *Schema) MarshalYAMLInline() (interface{}, error) {
	nb := high.NewNodeBuilder(s, s.low)
	nb.Resolve = true
	return nb.Render(), nil
}

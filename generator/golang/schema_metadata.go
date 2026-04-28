// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"encoding/json"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

type SchemaMetadataProvider interface {
	OpenAPISchemaMetadata() any
}

type providerSchemaMetadata struct {
	Ref                   string
	SchemaTypeRef         string
	ExclusiveMaximum      *providerDynamicBoolNumber
	ExclusiveMinimum      *providerDynamicBoolNumber
	Type                  []string
	AllOf                 []*providerSchemaMetadata
	OneOf                 []*providerSchemaMetadata
	AnyOf                 []*providerSchemaMetadata
	Discriminator         *providerDiscriminatorMetadata
	Examples              []*providerYAMLNode
	PrefixItems           []*providerSchemaMetadata
	Contains              *providerSchemaMetadata
	MinContains           *providerInt
	MaxContains           *providerInt
	If                    *providerSchemaMetadata
	Else                  *providerSchemaMetadata
	Then                  *providerSchemaMetadata
	DependentSchemas      []providerNamedSchemaMetadata
	DependentRequired     []providerStringList
	PatternProperties     []providerNamedSchemaMetadata
	PropertyNames         *providerSchemaMetadata
	UnevaluatedItems      *providerSchemaMetadata
	UnevaluatedProperties *providerDynamicSchemaBool
	Items                 *providerDynamicSchemaBool
	ID                    string
	Anchor                string
	DynamicAnchor         string
	DynamicRef            string
	Comment               string
	ContentSchema         *providerSchemaMetadata
	Vocabulary            []providerStringBool
	Not                   *providerSchemaMetadata
	Properties            []providerNamedSchemaMetadata
	Title                 string
	MultipleOf            *providerFloat
	Maximum               *providerFloat
	Minimum               *providerFloat
	MaxLength             *providerInt
	MinLength             *providerInt
	Pattern               string
	Format                string
	MaxItems              *providerInt
	MinItems              *providerInt
	UniqueItems           *providerBool
	MaxProperties         *providerInt
	MinProperties         *providerInt
	Required              []string
	Enum                  []*providerYAMLNode
	AdditionalProperties  *providerDynamicSchemaBool
	Description           string
	ContentEncoding       string
	ContentMediaType      string
	Default               *providerYAMLNode
	Const                 *providerYAMLNode
	Nullable              *providerBool
	ReadOnly              *providerBool
	WriteOnly             *providerBool
	Example               *providerYAMLNode
	Deprecated            *providerBool
	Extensions            []providerNamedYAMLNode
}

type providerDynamicBoolNumber struct {
	Bool   *providerBool
	Number *providerFloat
}

type providerDynamicSchemaBool struct {
	Schema *providerSchemaMetadata
	Bool   *providerBool
}

type providerDiscriminatorMetadata struct {
	PropertyName   string
	Mapping        []providerStringString
	DefaultMapping string
}

type providerNamedSchemaMetadata struct {
	Name   string
	Schema *providerSchemaMetadata
}

type providerNamedYAMLNode struct {
	Name  string
	Value *providerYAMLNode
}

type providerStringBool struct {
	Name  string
	Value bool
}

type providerStringString struct {
	Name  string
	Value string
}

type providerStringList struct {
	Name   string
	Values []string
}

type providerYAMLNode struct {
	Kind    string
	Style   int
	Tag     string
	Value   string
	Anchor  string
	Content []*providerYAMLNode
	Alias   *providerYAMLNode
}

type providerFloat struct {
	Value float64
}

type providerInt struct {
	Value int64
}

type providerBool struct {
	Value bool
}

func schemaProxyFromProviderMetadata(value any) (*highbase.SchemaProxy, error) {
	if value == nil {
		return nil, ErrNilSchema
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var metadata providerSchemaMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	return schemaProxyFromMetadata(&metadata), nil
}

func schemaProxyFromMetadata(metadata *providerSchemaMetadata) *highbase.SchemaProxy {
	if metadata == nil {
		return nil
	}
	if metadata.Ref != "" {
		if schemaMetadataHasSiblings(metadata) {
			return highbase.CreateSchemaProxyRefWithSchema(metadata.Ref, schemaFromMetadata(metadata))
		}
		return highbase.CreateSchemaProxyRef(metadata.Ref)
	}
	return highbase.CreateSchemaProxy(schemaFromMetadata(metadata))
}

func schemaFromMetadata(metadata *providerSchemaMetadata) *highbase.Schema {
	if metadata == nil {
		return nil
	}
	return &highbase.Schema{
		SchemaTypeRef:         metadata.SchemaTypeRef,
		ExclusiveMaximum:      dynamicBoolNumberFromMetadata(metadata.ExclusiveMaximum),
		ExclusiveMinimum:      dynamicBoolNumberFromMetadata(metadata.ExclusiveMinimum),
		Type:                  append([]string(nil), metadata.Type...),
		AllOf:                 schemaSliceFromMetadata(metadata.AllOf),
		OneOf:                 schemaSliceFromMetadata(metadata.OneOf),
		AnyOf:                 schemaSliceFromMetadata(metadata.AnyOf),
		Discriminator:         discriminatorFromMetadata(metadata.Discriminator),
		Examples:              yamlNodeSliceFromMetadata(metadata.Examples),
		PrefixItems:           schemaSliceFromMetadata(metadata.PrefixItems),
		Contains:              schemaProxyFromMetadata(metadata.Contains),
		MinContains:           intFromMetadata(metadata.MinContains),
		MaxContains:           intFromMetadata(metadata.MaxContains),
		If:                    schemaProxyFromMetadata(metadata.If),
		Else:                  schemaProxyFromMetadata(metadata.Else),
		Then:                  schemaProxyFromMetadata(metadata.Then),
		DependentSchemas:      schemaMapFromMetadata(metadata.DependentSchemas),
		DependentRequired:     stringListMapFromMetadata(metadata.DependentRequired),
		PatternProperties:     schemaMapFromMetadata(metadata.PatternProperties),
		PropertyNames:         schemaProxyFromMetadata(metadata.PropertyNames),
		UnevaluatedItems:      schemaProxyFromMetadata(metadata.UnevaluatedItems),
		UnevaluatedProperties: dynamicSchemaBoolFromMetadata(metadata.UnevaluatedProperties),
		Items:                 dynamicSchemaBoolFromMetadata(metadata.Items),
		Id:                    metadata.ID,
		Anchor:                metadata.Anchor,
		DynamicAnchor:         metadata.DynamicAnchor,
		DynamicRef:            metadata.DynamicRef,
		Comment:               metadata.Comment,
		ContentSchema:         schemaProxyFromMetadata(metadata.ContentSchema),
		Vocabulary:            stringBoolMapFromMetadata(metadata.Vocabulary),
		Not:                   schemaProxyFromMetadata(metadata.Not),
		Properties:            schemaMapFromMetadata(metadata.Properties),
		Title:                 metadata.Title,
		MultipleOf:            floatFromMetadata(metadata.MultipleOf),
		Maximum:               floatFromMetadata(metadata.Maximum),
		Minimum:               floatFromMetadata(metadata.Minimum),
		MaxLength:             intFromMetadata(metadata.MaxLength),
		MinLength:             intFromMetadata(metadata.MinLength),
		Pattern:               metadata.Pattern,
		Format:                metadata.Format,
		MaxItems:              intFromMetadata(metadata.MaxItems),
		MinItems:              intFromMetadata(metadata.MinItems),
		UniqueItems:           boolFromMetadata(metadata.UniqueItems),
		MaxProperties:         intFromMetadata(metadata.MaxProperties),
		MinProperties:         intFromMetadata(metadata.MinProperties),
		Required:              append([]string(nil), metadata.Required...),
		Enum:                  yamlNodeSliceFromMetadata(metadata.Enum),
		AdditionalProperties:  dynamicSchemaBoolFromMetadata(metadata.AdditionalProperties),
		Description:           metadata.Description,
		ContentEncoding:       metadata.ContentEncoding,
		ContentMediaType:      metadata.ContentMediaType,
		Default:               yamlNodeFromMetadata(metadata.Default),
		Const:                 yamlNodeFromMetadata(metadata.Const),
		Nullable:              boolFromMetadata(metadata.Nullable),
		ReadOnly:              boolFromMetadata(metadata.ReadOnly),
		WriteOnly:             boolFromMetadata(metadata.WriteOnly),
		Example:               yamlNodeFromMetadata(metadata.Example),
		Deprecated:            boolFromMetadata(metadata.Deprecated),
		Extensions:            extensionsFromMetadata(metadata.Extensions),
	}
}

func schemaMetadataHasSiblings(metadata *providerSchemaMetadata) bool {
	if metadata == nil {
		return false
	}
	cp := *metadata
	cp.Ref = ""
	return !schemaMetadataEmpty(&cp)
}

func schemaMetadataEmpty(metadata *providerSchemaMetadata) bool {
	return metadata == nil ||
		(metadata.SchemaTypeRef == "" && metadata.ExclusiveMaximum == nil && metadata.ExclusiveMinimum == nil &&
			len(metadata.Type) == 0 && len(metadata.AllOf) == 0 && len(metadata.OneOf) == 0 && len(metadata.AnyOf) == 0 &&
			metadata.Discriminator == nil && len(metadata.Examples) == 0 && len(metadata.PrefixItems) == 0 &&
			metadata.Contains == nil && metadata.MinContains == nil && metadata.MaxContains == nil && metadata.If == nil &&
			metadata.Else == nil && metadata.Then == nil && len(metadata.DependentSchemas) == 0 &&
			len(metadata.DependentRequired) == 0 && len(metadata.PatternProperties) == 0 && metadata.PropertyNames == nil &&
			metadata.UnevaluatedItems == nil && metadata.UnevaluatedProperties == nil && metadata.Items == nil &&
			metadata.ID == "" && metadata.Anchor == "" && metadata.DynamicAnchor == "" && metadata.DynamicRef == "" &&
			metadata.Comment == "" && metadata.ContentSchema == nil && len(metadata.Vocabulary) == 0 && metadata.Not == nil &&
			len(metadata.Properties) == 0 && metadata.Title == "" && metadata.MultipleOf == nil && metadata.Maximum == nil &&
			metadata.Minimum == nil && metadata.MaxLength == nil && metadata.MinLength == nil && metadata.Pattern == "" &&
			metadata.Format == "" && metadata.MaxItems == nil && metadata.MinItems == nil && metadata.UniqueItems == nil &&
			metadata.MaxProperties == nil && metadata.MinProperties == nil && len(metadata.Required) == 0 &&
			len(metadata.Enum) == 0 && metadata.AdditionalProperties == nil && metadata.Description == "" &&
			metadata.ContentEncoding == "" && metadata.ContentMediaType == "" && metadata.Default == nil &&
			metadata.Const == nil && metadata.Nullable == nil && metadata.ReadOnly == nil && metadata.WriteOnly == nil &&
			metadata.Example == nil && metadata.Deprecated == nil && len(metadata.Extensions) == 0)
}

func dynamicBoolNumberFromMetadata(metadata *providerDynamicBoolNumber) *highbase.DynamicValue[bool, float64] {
	if metadata == nil {
		return nil
	}
	if metadata.Number != nil {
		return &highbase.DynamicValue[bool, float64]{N: 1, B: metadata.Number.Value}
	}
	if metadata.Bool != nil {
		return &highbase.DynamicValue[bool, float64]{A: metadata.Bool.Value}
	}
	return nil
}

func dynamicSchemaBoolFromMetadata(metadata *providerDynamicSchemaBool) *highbase.DynamicValue[*highbase.SchemaProxy, bool] {
	if metadata == nil {
		return nil
	}
	if metadata.Bool != nil {
		return &highbase.DynamicValue[*highbase.SchemaProxy, bool]{N: 1, B: metadata.Bool.Value}
	}
	return &highbase.DynamicValue[*highbase.SchemaProxy, bool]{A: schemaProxyFromMetadata(metadata.Schema)}
}

func discriminatorFromMetadata(metadata *providerDiscriminatorMetadata) *highbase.Discriminator {
	if metadata == nil {
		return nil
	}
	return &highbase.Discriminator{
		PropertyName:   metadata.PropertyName,
		Mapping:        stringStringMapFromMetadata(metadata.Mapping),
		DefaultMapping: metadata.DefaultMapping,
	}
}

func schemaSliceFromMetadata(values []*providerSchemaMetadata) []*highbase.SchemaProxy {
	if len(values) == 0 {
		return nil
	}
	out := make([]*highbase.SchemaProxy, 0, len(values))
	for _, value := range values {
		out = append(out, schemaProxyFromMetadata(value))
	}
	return out
}

func schemaMapFromMetadata(values []providerNamedSchemaMetadata) *orderedmap.Map[string, *highbase.SchemaProxy] {
	if len(values) == 0 {
		return nil
	}
	out := orderedmap.New[string, *highbase.SchemaProxy]()
	for _, value := range values {
		out.Set(value.Name, schemaProxyFromMetadata(value.Schema))
	}
	return out
}

func stringBoolMapFromMetadata(values []providerStringBool) *orderedmap.Map[string, bool] {
	if len(values) == 0 {
		return nil
	}
	out := orderedmap.New[string, bool]()
	for _, value := range values {
		out.Set(value.Name, value.Value)
	}
	return out
}

func stringStringMapFromMetadata(values []providerStringString) *orderedmap.Map[string, string] {
	if len(values) == 0 {
		return nil
	}
	out := orderedmap.New[string, string]()
	for _, value := range values {
		out.Set(value.Name, value.Value)
	}
	return out
}

func stringListMapFromMetadata(values []providerStringList) *orderedmap.Map[string, []string] {
	if len(values) == 0 {
		return nil
	}
	out := orderedmap.New[string, []string]()
	for _, value := range values {
		out.Set(value.Name, append([]string(nil), value.Values...))
	}
	return out
}

func extensionsFromMetadata(values []providerNamedYAMLNode) *orderedmap.Map[string, *yaml.Node] {
	if len(values) == 0 {
		return nil
	}
	out := orderedmap.New[string, *yaml.Node]()
	for _, value := range values {
		out.Set(value.Name, yamlNodeFromMetadata(value.Value))
	}
	return out
}

func yamlNodeSliceFromMetadata(values []*providerYAMLNode) []*yaml.Node {
	if len(values) == 0 {
		return nil
	}
	out := make([]*yaml.Node, 0, len(values))
	for _, value := range values {
		out = append(out, yamlNodeFromMetadata(value))
	}
	return out
}

func yamlNodeFromMetadata(metadata *providerYAMLNode) *yaml.Node {
	if metadata == nil {
		return nil
	}
	node := &yaml.Node{
		Kind:   yamlKindFromMetadata(metadata.Kind),
		Style:  yaml.Style(metadata.Style),
		Tag:    metadata.Tag,
		Value:  metadata.Value,
		Anchor: metadata.Anchor,
		Alias:  yamlNodeFromMetadata(metadata.Alias),
	}
	for _, child := range metadata.Content {
		node.Content = append(node.Content, yamlNodeFromMetadata(child))
	}
	return node
}

func yamlKindFromMetadata(kind string) yaml.Kind {
	switch kind {
	case "document":
		return yaml.DocumentNode
	case "sequence":
		return yaml.SequenceNode
	case "mapping":
		return yaml.MappingNode
	case "alias":
		return yaml.AliasNode
	default:
		return yaml.ScalarNode
	}
}

func floatFromMetadata(metadata *providerFloat) *float64 {
	if metadata == nil {
		return nil
	}
	value := metadata.Value
	return &value
}

func intFromMetadata(metadata *providerInt) *int64 {
	if metadata == nil {
		return nil
	}
	value := metadata.Value
	return &value
}

func boolFromMetadata(metadata *providerBool) *bool {
	if metadata == nil {
		return nil
	}
	value := metadata.Value
	return &value
}

// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

// Constants for labels used to look up values within OpenAPI specifications.
const (
	TagsLabel                  = "tags"
	ExternalDocsLabel          = "externalDocs"
	ExamplesLabel              = "examples"
	ExampleLabel               = "example"
	ValueLabel                 = "value"
	InfoLabel                  = "info"
	ContactLabel               = "contact"
	LicenseLabel               = "license"
	PropertiesLabel            = "properties"
	DependentSchemasLabel      = "dependentSchemas"
	PatternPropertiesLabel     = "patternProperties"
	IfLabel                    = "if"
	ElseLabel                  = "else"
	ThenLabel                  = "then"
	PropertyNamesLabel         = "propertyNames"
	UnevaluatedItemsLabel      = "unevaluatedItems"
	UnevaluatedPropertiesLabel = "unevaluatedProperties"
	AdditionalPropertiesLabel  = "additionalProperties"
	XMLLabel                   = "xml"
	ItemsLabel                 = "items"
	PrefixItemsLabel           = "prefixItems"
	ContainsLabel              = "contains"
	AllOfLabel                 = "allOf"
	AnyOfLabel                 = "anyOf"
	OneOfLabel                 = "oneOf"
	NotLabel                   = "not"
	TypeLabel                  = "type"
	DiscriminatorLabel        = "discriminator"
	ExclusiveMinimumLabel     = "exclusiveMinimum"
	ExclusiveMaximumLabel     = "exclusiveMaximum"
	SchemaLabel               = "schema"
	SchemaTypeLabel           = "$schema"
)

/*
PropertyNames         low.NodeReference[*SchemaProxy]
			UnevaluatedItems      low.NodeReference[*SchemaProxy]
			UnevaluatedProperties low.NodeReference[*SchemaProxy]
*/
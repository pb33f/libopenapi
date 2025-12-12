// Copyright 2022-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultBreakingRules(t *testing.T) {
	config := GenerateDefaultBreakingRules()
	assert.NotNil(t, config)

	// test top-level rules
	assert.True(t, *config.OpenAPI.Added)
	assert.True(t, *config.OpenAPI.Modified)
	assert.True(t, *config.OpenAPI.Removed)
	assert.True(t, *config.JSONSchemaDialect.Added)
}

func TestDefaultBreakingRules_Info(t *testing.T) {
	config := GenerateDefaultBreakingRules()
	assert.NotNil(t, config.Info)

	// info properties are non-breaking
	assert.False(t, *config.Info.Title.Added)
	assert.False(t, *config.Info.Title.Modified)
	assert.False(t, *config.Info.Title.Removed)
	assert.False(t, *config.Info.Summary.Modified)
	assert.False(t, *config.Info.Description.Modified)
	assert.False(t, *config.Info.TermsOfService.Modified)
	assert.False(t, *config.Info.Version.Modified)

	// contact and license sub-object changes are non-breaking
	assert.False(t, *config.Info.Contact.Added)
	assert.False(t, *config.Info.Contact.Modified)
	assert.False(t, *config.Info.Contact.Removed)
	assert.False(t, *config.Info.License.Added)
	assert.False(t, *config.Info.License.Modified)
	assert.False(t, *config.Info.License.Removed)
}

func TestDefaultBreakingRules_PathItem(t *testing.T) {
	config := GenerateDefaultBreakingRules()
	assert.NotNil(t, config.PathItem)

	// HTTP methods: adding is fine, removing is breaking
	assert.False(t, *config.PathItem.Get.Added)
	assert.True(t, *config.PathItem.Get.Removed)
	assert.False(t, *config.PathItem.Post.Added)
	assert.True(t, *config.PathItem.Post.Removed)
	assert.False(t, *config.PathItem.Put.Added)
	assert.True(t, *config.PathItem.Put.Removed)
	assert.False(t, *config.PathItem.Delete.Added)
	assert.True(t, *config.PathItem.Delete.Removed)
}

func TestDefaultBreakingRules_Schema(t *testing.T) {
	config := GenerateDefaultBreakingRules()
	assert.NotNil(t, config.Schema)

	// type modifications are breaking
	assert.True(t, *config.Schema.Type.Modified)

	// description changes are not breaking
	assert.False(t, *config.Schema.Description.Modified)

	// required property changes
	assert.True(t, *config.Schema.Required.Added)
	assert.True(t, *config.Schema.Required.Removed)

	// enum removals are breaking, additions are not
	assert.False(t, *config.Schema.Enum.Added)
	assert.True(t, *config.Schema.Enum.Removed)
}

func TestDefaultBreakingRules_Operation(t *testing.T) {
	config := GenerateDefaultBreakingRules()
	assert.NotNil(t, config.Operation)

	// operationId changes are breaking
	assert.True(t, *config.Operation.OperationID.Added)
	assert.True(t, *config.Operation.OperationID.Modified)
	assert.True(t, *config.Operation.OperationID.Removed)

	// summary/description changes are not breaking
	assert.False(t, *config.Operation.Summary.Modified)
	assert.False(t, *config.Operation.Description.Modified)

	// request body: adding is breaking, removing is breaking
	assert.True(t, *config.Operation.RequestBody.Added)
	assert.True(t, *config.Operation.RequestBody.Removed)
}

func TestMerge_NilOverride(t *testing.T) {
	ResetDefaultBreakingRules()
	defer ResetDefaultBreakingRules()

	config := GenerateDefaultBreakingRules()
	original := *config.Info.Title.Modified
	config.Merge(nil)

	// should not change anything
	assert.Equal(t, original, *config.Info.Title.Modified)
}

func TestMerge_SinglePropertyOverride(t *testing.T) {
	ResetDefaultBreakingRules()
	defer ResetDefaultBreakingRules()

	config := GenerateDefaultBreakingRules()

	// verify default
	assert.False(t, *config.Info.Description.Modified)

	// override to make description changes breaking
	override := &BreakingRulesConfig{
		Info: &InfoRules{
			Description: &BreakingChangeRule{
				Modified: boolPtr(true),
			},
		},
	}
	config.Merge(override)

	// description is now breaking
	assert.True(t, *config.Info.Description.Modified)

	// other info fields unchanged
	assert.False(t, *config.Info.Title.Modified)
	assert.False(t, *config.Info.Summary.Modified)
}

func TestMerge_MultipleOverrides(t *testing.T) {
	ResetDefaultBreakingRules()
	defer ResetDefaultBreakingRules()

	config := GenerateDefaultBreakingRules()

	override := &BreakingRulesConfig{
		Operation: &OperationRules{
			OperationID: &BreakingChangeRule{
				Modified: boolPtr(false), // don't consider operationId changes breaking
			},
		},
		Schema: &SchemaRules{
			Format: &BreakingChangeRule{
				Modified: boolPtr(false), // don't consider format changes breaking
			},
		},
	}
	config.Merge(override)

	assert.False(t, *config.Operation.OperationID.Modified)
	assert.False(t, *config.Schema.Format.Modified)

	// other values unchanged
	assert.True(t, *config.Operation.OperationID.Added)
	assert.True(t, *config.Operation.OperationID.Removed)
	assert.True(t, *config.Schema.Type.Modified)
}

func TestMerge_PartialRuleOverride(t *testing.T) {
	ResetDefaultBreakingRules()
	defer ResetDefaultBreakingRules()

	config := GenerateDefaultBreakingRules()

	// only override the 'modified' aspect, leave added/removed unchanged
	override := &BreakingRulesConfig{
		Schema: &SchemaRules{
			Required: &BreakingChangeRule{
				Modified: boolPtr(true), // only set modified
			},
		},
	}
	config.Merge(override)

	// modified is now set
	assert.True(t, *config.Schema.Required.Modified)

	// added and removed remain at their defaults
	assert.True(t, *config.Schema.Required.Added)
	assert.True(t, *config.Schema.Required.Removed)
}

func TestMerge_TopLevelOverride(t *testing.T) {
	ResetDefaultBreakingRules()
	defer ResetDefaultBreakingRules()

	config := GenerateDefaultBreakingRules()

	override := &BreakingRulesConfig{
		OpenAPI: &BreakingChangeRule{
			Modified: boolPtr(false),
		},
	}
	config.Merge(override)

	assert.False(t, *config.OpenAPI.Modified)
	assert.True(t, *config.OpenAPI.Added) // unchanged
}

func TestMerge_NilBaseComponent(t *testing.T) {
	// start with an empty config
	config := &BreakingRulesConfig{}

	override := &BreakingRulesConfig{
		Info: &InfoRules{
			Title: &BreakingChangeRule{
				Modified: boolPtr(true),
			},
		},
	}
	config.Merge(override)

	assert.NotNil(t, config.Info)
	assert.NotNil(t, config.Info.Title)
	assert.True(t, *config.Info.Title.Modified)
}

func TestIsBreaking_Found(t *testing.T) {
	config := GenerateDefaultBreakingRules()

	// schema type modified is breaking
	assert.True(t, config.IsBreaking("schema", "type", ChangeTypeModified))

	// schema description modified is not breaking
	assert.False(t, config.IsBreaking("schema", "description", ChangeTypeModified))

	// path item get removed is breaking
	assert.True(t, config.IsBreaking("pathItem", "get", ChangeTypeRemoved))

	// path item get added is not breaking
	assert.False(t, config.IsBreaking("pathItem", "get", ChangeTypeAdded))
}

func TestIsBreaking_NotFound(t *testing.T) {
	config := GenerateDefaultBreakingRules()

	// unknown component
	assert.False(t, config.IsBreaking("unknown", "property", ChangeTypeModified))

	// unknown property
	assert.False(t, config.IsBreaking("schema", "unknownProperty", ChangeTypeModified))
}

func TestIsBreaking_EmptyConfig(t *testing.T) {
	config := &BreakingRulesConfig{}

	// returns false when no rules defined
	assert.False(t, config.IsBreaking("schema", "type", ChangeTypeModified))
}

func TestIsBreaking_InvalidChangeType(t *testing.T) {
	config := GenerateDefaultBreakingRules()

	// unknown change type returns false
	assert.False(t, config.IsBreaking("schema", "type", "invalid"))
}

func TestGetRule_AllComponents(t *testing.T) {
	config := GenerateDefaultBreakingRules()

	// test that all components return non-nil rules
	tests := []struct {
		component string
		property  string
	}{
		{"openapi", ""},
		{"jsonSchemaDialect", ""},
		{"info", "title"},
		{"contact", "url"},
		{"license", "name"},
		{"paths", "path"},
		{"pathItem", "get"},
		{"operation", "operationId"},
		{"parameter", "name"},
		{"requestBody", "required"},
		{"responses", "default"},
		{"response", "description"},
		{"mediaType", "schema"},
		{"encoding", "contentType"},
		{"header", "required"},
		{"schema", "type"},
		{"discriminator", "propertyName"},
		{"xml", "name"},
		{"server", "url"},
		{"serverVariable", "default"},
		{"tag", "name"},
		{"externalDocs", "url"},
		{"securityScheme", "type"},
		{"securityRequirement", "schemes"},
		{"oauthFlows", "implicit"},
		{"oauthFlow", "authorizationUrl"},
		{"callback", "expressions"},
		{"link", "operationRef"},
		{"example", "summary"},
	}

	for _, tt := range tests {
		t.Run(tt.component+"/"+tt.property, func(t *testing.T) {
			rule := config.GetRule(tt.component, tt.property)
			assert.NotNil(t, rule, "expected rule for %s/%s", tt.component, tt.property)
		})
	}
}

func TestGetRule_SchemaProperties(t *testing.T) {
	config := GenerateDefaultBreakingRules()

	// test all schema properties
	schemaProperties := []string{
		"$ref", "type", "title", "description", "format",
		"maximum", "minimum", "exclusiveMaximum", "exclusiveMinimum",
		"maxLength", "minLength", "pattern", "maxItems", "minItems",
		"maxProperties", "minProperties", "uniqueItems", "multipleOf",
		"contentEncoding", "contentMediaType", "default", "const",
		"nullable", "readOnly", "writeOnly", "deprecated", "example",
		"required", "enum", "properties", "additionalProperties",
		"allOf", "anyOf", "oneOf", "prefixItems", "items",
		"discriminator", "externalDocs", "not", "if", "then", "else",
		"propertyNames", "contains", "unevaluatedItems", "unevaluatedProperties",
		"dependentRequired",
	}

	for _, prop := range schemaProperties {
		t.Run("schema/"+prop, func(t *testing.T) {
			rule := config.GetRule("schema", prop)
			assert.NotNil(t, rule, "expected rule for schema/%s", prop)
		})
	}
}

func TestMergeRule_BothNil(t *testing.T) {
	result := mergeRule(nil, nil)
	assert.Nil(t, result)
}

func TestMergeRule_BaseNil(t *testing.T) {
	override := &BreakingChangeRule{
		Added: boolPtr(true),
	}
	result := mergeRule(nil, override)
	assert.Equal(t, override, result)
}

func TestMergeRule_OverrideNil(t *testing.T) {
	base := &BreakingChangeRule{
		Added: boolPtr(true),
	}
	result := mergeRule(base, nil)
	assert.Equal(t, base, result)
}

func TestHelperFunctions(t *testing.T) {
	// test boolPtr
	p := boolPtr(true)
	assert.NotNil(t, p)
	assert.True(t, *p)

	p = boolPtr(false)
	assert.NotNil(t, p)
	assert.False(t, *p)

	// test rule helper
	r := rule(true, false, true)
	assert.NotNil(t, r)
	assert.True(t, *r.Added)
	assert.False(t, *r.Modified)
	assert.True(t, *r.Removed)
}

func TestMerge_AllComponents(t *testing.T) {
	ResetDefaultBreakingRules()
	defer ResetDefaultBreakingRules()

	config := GenerateDefaultBreakingRules()

	// create override with all components
	override := &BreakingRulesConfig{
		OpenAPI:             rule(false, false, false),
		JSONSchemaDialect:   rule(false, false, false),
		Info:                &InfoRules{Title: rule(true, true, true)},
		Contact:             &ContactRules{URL: rule(true, true, true)},
		License:             &LicenseRules{Name: rule(true, true, true)},
		Paths:               &PathsRules{Path: rule(true, true, false)},
		PathItem:            &PathItemRules{Get: rule(true, true, false)},
		Operation:           &OperationRules{OperationID: rule(false, false, false)},
		Parameter:           &ParameterRules{Name: rule(false, false, false)},
		RequestBody:         &RequestBodyRules{Required: rule(false, false, false)},
		Responses:           &ResponsesRules{Default: rule(true, true, false)},
		Response:            &ResponseRules{Description: rule(true, true, true)},
		MediaType:           &MediaTypeRules{Example: rule(true, true, true)},
		Encoding:            &EncodingRules{ContentType: rule(false, false, false)},
		Header:              &HeaderRules{Required: rule(false, false, false)},
		Schema:              &SchemaRules{Type: rule(false, false, false)},
		Discriminator:       &DiscriminatorRules{PropertyName: rule(false, false, false)},
		XML:                 &XMLRules{Name: rule(false, false, false)},
		Server:              &ServerRules{URL: rule(false, false, false)},
		ServerVariable:      &ServerVariableRules{Default: rule(false, false, false)},
		Tag:                 &TagRules{Name: rule(false, false, false)},
		ExternalDocs:        &ExternalDocsRules{URL: rule(true, true, true)},
		SecurityScheme:      &SecuritySchemeRules{Type: rule(false, false, false)},
		SecurityRequirement: &SecurityRequirementRules{Schemes: rule(true, true, false)},
		OAuthFlows:          &OAuthFlowsRules{Implicit: rule(true, true, false)},
		OAuthFlow:           &OAuthFlowRules{AuthorizationURL: rule(false, false, false)},
		Callback:            &CallbackRules{Expressions: rule(true, true, false)},
		Link:                &LinkRules{OperationRef: rule(false, false, false)},
		Example:             &ExampleRules{Summary: rule(true, true, true)},
	}

	config.Merge(override)

	// verify overrides applied
	assert.False(t, *config.OpenAPI.Added)
	assert.True(t, *config.Info.Title.Added)
	assert.True(t, *config.Contact.URL.Added)
	assert.False(t, *config.Operation.OperationID.Added)
	assert.False(t, *config.Schema.Type.Modified)
}

func TestIsBreaking_ChangeTypes(t *testing.T) {
	config := GenerateDefaultBreakingRules()

	tests := []struct {
		component  string
		property   string
		changeType string
		expected   bool
	}{
		// schema type
		{"schema", "type", ChangeTypeAdded, false},
		{"schema", "type", ChangeTypeModified, true},
		{"schema", "type", ChangeTypeRemoved, false},

		// schema required
		{"schema", "required", ChangeTypeAdded, true},
		{"schema", "required", ChangeTypeRemoved, true},

		// enum
		{"schema", "enum", ChangeTypeAdded, false},
		{"schema", "enum", ChangeTypeRemoved, true},

		// path removal
		{"paths", "path", ChangeTypeRemoved, true},
		{"paths", "path", ChangeTypeAdded, false},

		// operation descriptions
		{"operation", "description", ChangeTypeAdded, false},
		{"operation", "description", ChangeTypeModified, false},
		{"operation", "description", ChangeTypeRemoved, false},
	}

	for _, tt := range tests {
		name := tt.component + "/" + tt.property + "/" + tt.changeType
		t.Run(name, func(t *testing.T) {
			result := config.IsBreaking(tt.component, tt.property, tt.changeType)
			assert.Equal(t, tt.expected, result, "IsBreaking(%s, %s, %s)", tt.component, tt.property, tt.changeType)
		})
	}
}

func TestDefaultBreakingRules_NilRuleValue(t *testing.T) {
	config := GenerateDefaultBreakingRules()

	// ensure no rule has nil Added/Modified/Removed
	// checking a few critical ones
	assert.NotNil(t, config.Schema.Type.Added)
	assert.NotNil(t, config.Schema.Type.Modified)
	assert.NotNil(t, config.Schema.Type.Removed)

	assert.NotNil(t, config.PathItem.Get.Added)
	assert.NotNil(t, config.PathItem.Get.Removed)

	assert.NotNil(t, config.Operation.RequestBody.Added)
	assert.NotNil(t, config.Operation.RequestBody.Removed)
}

func TestMerge_EmptyOverrideComponent(t *testing.T) {
	ResetDefaultBreakingRules()
	defer ResetDefaultBreakingRules()

	config := GenerateDefaultBreakingRules()
	originalValue := *config.Schema.Type.Modified

	// merge with empty schema rules (no properties set)
	override := &BreakingRulesConfig{
		Schema: &SchemaRules{},
	}
	config.Merge(override)

	// should not change anything since override has no values
	assert.Equal(t, originalValue, *config.Schema.Type.Modified)
}

func TestMerge_InvalidatesCacheAfterMerge(t *testing.T) {
	// create fresh config and access GetRule to populate cache
	config := &BreakingRulesConfig{
		Schema: &SchemaRules{
			Type: &BreakingChangeRule{
				Modified: boolPtr(false),
			},
		},
	}

	// trigger cache population
	rule := config.GetRule("schema", "type")
	assert.NotNil(t, rule)
	assert.False(t, *rule.Modified)

	// merge an override that changes the value
	override := &BreakingRulesConfig{
		Schema: &SchemaRules{
			Type: &BreakingChangeRule{
				Modified: boolPtr(true),
			},
		},
	}
	config.Merge(override)

	// cache should be invalidated and rebuilt, returning new value
	rule = config.GetRule("schema", "type")
	assert.NotNil(t, rule)
	assert.True(t, *rule.Modified) // should reflect merged value
}

func TestGetRule_UnknownComponent(t *testing.T) {
	config := GenerateDefaultBreakingRules()

	rule := config.GetRule("notAComponent", "someProperty")
	assert.Nil(t, rule)
}

func TestGetRule_UnknownProperty(t *testing.T) {
	config := GenerateDefaultBreakingRules()

	// known component, unknown property
	rule := config.GetRule("schema", "notAProperty")
	assert.Nil(t, rule)

	rule = config.GetRule("info", "notAProperty")
	assert.Nil(t, rule)
}

func TestIsBreaking_NilRuleField(t *testing.T) {
	// create config with partial rule (some fields nil)
	config := &BreakingRulesConfig{
		Schema: &SchemaRules{
			Type: &BreakingChangeRule{
				Modified: boolPtr(true),
				// Added and Removed are nil
			},
		},
	}

	// modified is set
	assert.True(t, config.IsBreaking("schema", "type", ChangeTypeModified))

	// added and removed return false when nil
	assert.False(t, config.IsBreaking("schema", "type", ChangeTypeAdded))
	assert.False(t, config.IsBreaking("schema", "type", ChangeTypeRemoved))
}

func TestDefaultBreakingRules_Singleton(t *testing.T) {
	ResetDefaultBreakingRules()
	defer ResetDefaultBreakingRules()

	// first call creates the singleton
	config1 := GenerateDefaultBreakingRules()
	assert.NotNil(t, config1)

	// second call returns the same instance
	config2 := GenerateDefaultBreakingRules()
	assert.Same(t, config1, config2)
}

func TestDefaultBreakingRules_Concurrent(t *testing.T) {
	ResetDefaultBreakingRules()
	defer ResetDefaultBreakingRules()

	var wg sync.WaitGroup
	configs := make([]*BreakingRulesConfig, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			configs[idx] = GenerateDefaultBreakingRules()
		}(i)
	}
	wg.Wait()

	// all should be the same instance
	first := configs[0]
	for i := 1; i < 100; i++ {
		assert.Same(t, first, configs[i])
	}
}

func TestResetDefaultBreakingRules(t *testing.T) {
	ResetDefaultBreakingRules()
	defer ResetDefaultBreakingRules()

	config1 := GenerateDefaultBreakingRules()

	ResetDefaultBreakingRules()

	config2 := GenerateDefaultBreakingRules()

	// after reset, we get a new instance
	assert.NotSame(t, config1, config2)
}

func BenchmarkDefaultBreakingRules(b *testing.B) {
	ResetDefaultBreakingRules()
	defer ResetDefaultBreakingRules()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateDefaultBreakingRules()
	}
}

func BenchmarkDefaultBreakingRules_FirstCall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ResetDefaultBreakingRules()
		_ = GenerateDefaultBreakingRules()
	}
}

func BenchmarkMerge(b *testing.B) {
	override := &BreakingRulesConfig{
		Schema: &SchemaRules{Type: rule(false, false, false)},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := GenerateDefaultBreakingRules()
		// create a copy since we can't mutate the singleton
		configCopy := *config
		configCopy.Merge(override)
	}
}

func TestMerge_InfoContactLicenseOverride(t *testing.T) {
	ResetDefaultBreakingRules()
	defer ResetDefaultBreakingRules()

	config := GenerateDefaultBreakingRules()

	// verify defaults are non-breaking
	assert.False(t, *config.Info.Contact.Added)
	assert.False(t, *config.Info.Contact.Removed)
	assert.False(t, *config.Info.License.Added)
	assert.False(t, *config.Info.License.Removed)

	// override to make contact/license changes breaking
	override := &BreakingRulesConfig{
		Info: &InfoRules{
			Contact: &BreakingChangeRule{
				Added:   boolPtr(true),
				Removed: boolPtr(true),
			},
			License: &BreakingChangeRule{
				Removed: boolPtr(true),
			},
		},
	}
	config.Merge(override)

	// contact changes are now breaking
	assert.True(t, *config.Info.Contact.Added)
	assert.True(t, *config.Info.Contact.Removed)

	// license removal is now breaking, but not addition
	assert.False(t, *config.Info.License.Added) // unchanged
	assert.True(t, *config.Info.License.Removed)

	// GetRule should work for info.contact and info.license
	contactRule := config.GetRule("info", "contact")
	assert.NotNil(t, contactRule)
	assert.True(t, *contactRule.Added)
	assert.True(t, *contactRule.Removed)

	licenseRule := config.GetRule("info", "license")
	assert.NotNil(t, licenseRule)
	assert.True(t, *licenseRule.Removed)
}

func TestSetActiveBreakingRulesConfig(t *testing.T) {
	// ensure clean state
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	// initially should return defaults
	config := GetActiveBreakingRulesConfig()
	assert.NotNil(t, config)
	assert.True(t, *config.Schema.Type.Modified) // default is true

	// create a standalone custom config (not merged from defaults to avoid shallow copy issues)
	customConfig := &BreakingRulesConfig{
		Schema: &SchemaRules{
			Type: &BreakingChangeRule{
				Added:    boolPtr(false),
				Modified: boolPtr(false),
				Removed:  boolPtr(false),
			},
		},
	}

	// set the custom config
	SetActiveBreakingRulesConfig(customConfig)

	// now should return custom config
	activeConfig := GetActiveBreakingRulesConfig()
	assert.NotNil(t, activeConfig)
	assert.False(t, *activeConfig.Schema.Type.Modified) // now false
}

func TestResetActiveBreakingRulesConfig(t *testing.T) {
	// ensure clean state
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	// create a standalone custom config (not merged from defaults to avoid shallow copy issues)
	customConfig := &BreakingRulesConfig{
		Schema: &SchemaRules{
			Type: &BreakingChangeRule{
				Added:    boolPtr(false),
				Modified: boolPtr(false),
				Removed:  boolPtr(false),
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	// verify custom config is active - Type.Modified should be false
	assert.False(t, *GetActiveBreakingRulesConfig().Schema.Type.Modified)

	// reset to defaults
	ResetActiveBreakingRulesConfig()

	// should return defaults again
	config := GetActiveBreakingRulesConfig()
	assert.True(t, *config.Schema.Type.Modified) // back to default true
}

func TestActiveConfigIntegration(t *testing.T) {
	// ensure clean state
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	// test that IsBreakingChange uses the active config
	assert.True(t, IsBreakingChange(CompSchema, PropType, ChangeTypeModified)) // default

	// create a standalone custom config (not merged from defaults to avoid shallow copy issues)
	customConfig := &BreakingRulesConfig{
		Schema: &SchemaRules{
			Type: &BreakingChangeRule{
				Added:    boolPtr(false),
				Modified: boolPtr(false),
				Removed:  boolPtr(false),
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	// now should use custom config
	assert.False(t, IsBreakingChange(CompSchema, PropType, ChangeTypeModified))

	// helper functions should also use active config
	assert.False(t, BreakingModified(CompSchema, PropType))

	// reset and verify
	ResetActiveBreakingRulesConfig()
	assert.True(t, BreakingModified(CompSchema, PropType))
}

func BenchmarkIsBreaking(b *testing.B) {
	config := GenerateDefaultBreakingRules()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.IsBreaking("schema", "type", ChangeTypeModified)
	}
}

func BenchmarkGetRule(b *testing.B) {
	config := GenerateDefaultBreakingRules()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.GetRule("schema", "type")
	}
}

func TestJsonTagName_NoOmitempty(t *testing.T) {
	// test the case where JSON tag has no comma (no omitempty)
	type testStruct struct {
		Field1 string `json:"field1"`
		Field2 string `json:"field2,omitempty"`
		Field3 string `json:"-"`
		Field4 string
	}

	typ := reflect.TypeOf(testStruct{})

	// field1 has no comma - should return "field1"
	name := jsonTagName(typ.Field(0))
	assert.Equal(t, "field1", name)

	// field2 has comma - should return "field2"
	name = jsonTagName(typ.Field(1))
	assert.Equal(t, "field2", name)

	// field3 has "-" - should return ""
	name = jsonTagName(typ.Field(2))
	assert.Equal(t, "", name)

	// field4 has no tag - should return ""
	name = jsonTagName(typ.Field(3))
	assert.Equal(t, "", name)
}

func TestMergeRulesStruct_NonRuleFields(t *testing.T) {
	// test the case where struct has non-rule fields that should be skipped
	type mixedStruct struct {
		RuleField    *BreakingChangeRule `json:"rule,omitempty"`
		NonRuleField string              `json:"nonrule,omitempty"`
	}

	base := mixedStruct{
		RuleField:    &BreakingChangeRule{Added: boolPtr(true)},
		NonRuleField: "base",
	}
	override := mixedStruct{
		RuleField:    &BreakingChangeRule{Added: boolPtr(false)},
		NonRuleField: "override",
	}

	baseVal := reflect.ValueOf(&base).Elem()
	overrideVal := reflect.ValueOf(&override).Elem()

	mergeRulesStruct(baseVal, overrideVal)

	// rule field should be merged
	assert.False(t, *base.RuleField.Added)
	// non-rule field should remain unchanged (not merged)
	assert.Equal(t, "base", base.NonRuleField)
}

func TestAddRulesToCache_NonRuleFields(t *testing.T) {
	// test the case where struct has fields that should be skipped
	type mixedStruct struct {
		RuleField    *BreakingChangeRule `json:"rule,omitempty"`
		NonRuleField string              `json:"nonrule,omitempty"`
		NoTagField   *BreakingChangeRule
	}

	data := mixedStruct{
		RuleField:    &BreakingChangeRule{Added: boolPtr(true)},
		NonRuleField: "test",
		NoTagField:   &BreakingChangeRule{Added: boolPtr(false)},
	}

	cache := make(map[string]*BreakingChangeRule)
	addRulesToCache(cache, "test", reflect.ValueOf(data))

	// only RuleField should be in cache (has proper tag and is rule type)
	assert.Len(t, cache, 1)
	assert.NotNil(t, cache["test.rule"])
	assert.True(t, *cache["test.rule"].Added)
	// NonRuleField and NoTagField should not be in cache
	assert.Nil(t, cache["test.nonrule"])
	assert.Nil(t, cache["test.NoTagField"])
}

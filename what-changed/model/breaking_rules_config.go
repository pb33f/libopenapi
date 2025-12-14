// Copyright 2022-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"reflect"
	"sync"
)

// BreakingRulesConfig holds all breaking change rules organized by OpenAPI component.
// Structure mirrors the OpenAPI 3.x specification.
type BreakingRulesConfig struct {
	OpenAPI             *BreakingChangeRule       `json:"openapi,omitempty" yaml:"openapi,omitempty"`
	JSONSchemaDialect   *BreakingChangeRule       `json:"jsonSchemaDialect,omitempty" yaml:"jsonSchemaDialect,omitempty"`
	Self                *BreakingChangeRule       `json:"$self,omitempty" yaml:"$self,omitempty"`
	Components          *BreakingChangeRule       `json:"components,omitempty" yaml:"components,omitempty"`
	Info                *InfoRules                `json:"info,omitempty" yaml:"info,omitempty"`
	Contact             *ContactRules             `json:"contact,omitempty" yaml:"contact,omitempty"`
	License             *LicenseRules             `json:"license,omitempty" yaml:"license,omitempty"`
	Paths               *PathsRules               `json:"paths,omitempty" yaml:"paths,omitempty"`
	PathItem            *PathItemRules            `json:"pathItem,omitempty" yaml:"pathItem,omitempty"`
	Operation           *OperationRules           `json:"operation,omitempty" yaml:"operation,omitempty"`
	Parameter           *ParameterRules           `json:"parameter,omitempty" yaml:"parameter,omitempty"`
	RequestBody         *RequestBodyRules         `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses           *ResponsesRules           `json:"responses,omitempty" yaml:"responses,omitempty"`
	Response            *ResponseRules            `json:"response,omitempty" yaml:"response,omitempty"`
	MediaType           *MediaTypeRules           `json:"mediaType,omitempty" yaml:"mediaType,omitempty"`
	Encoding            *EncodingRules            `json:"encoding,omitempty" yaml:"encoding,omitempty"`
	Header              *HeaderRules              `json:"header,omitempty" yaml:"header,omitempty"`
	Schema              *SchemaRules              `json:"schema,omitempty" yaml:"schema,omitempty"`
	Schemas             *BreakingChangeRule       `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Servers             *BreakingChangeRule       `json:"servers,omitempty" yaml:"servers,omitempty"`
	Discriminator       *DiscriminatorRules       `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`
	XML                 *XMLRules                 `json:"xml,omitempty" yaml:"xml,omitempty"`
	Server              *ServerRules              `json:"server,omitempty" yaml:"server,omitempty"`
	ServerVariable      *ServerVariableRules      `json:"serverVariable,omitempty" yaml:"serverVariable,omitempty"`
	Tags                *BreakingChangeRule       `json:"tags,omitempty" yaml:"tags,omitempty"`
	Tag                 *TagRules                 `json:"tag,omitempty" yaml:"tag,omitempty"`
	Security            *BreakingChangeRule       `json:"security,omitempty" yaml:"security,omitempty"`
	ExternalDocs        *ExternalDocsRules        `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	SecurityScheme      *SecuritySchemeRules      `json:"securityScheme,omitempty" yaml:"securityScheme,omitempty"`
	SecurityRequirement *SecurityRequirementRules `json:"securityRequirement,omitempty" yaml:"securityRequirement,omitempty"`
	OAuthFlows          *OAuthFlowsRules          `json:"oauthFlows,omitempty" yaml:"oauthFlows,omitempty"`
	OAuthFlow           *OAuthFlowRules           `json:"oauthFlow,omitempty" yaml:"oauthFlow,omitempty"`
	Callback            *CallbackRules            `json:"callback,omitempty" yaml:"callback,omitempty"`
	Link                *LinkRules                `json:"link,omitempty" yaml:"link,omitempty"`
	Example             *ExampleRules             `json:"example,omitempty" yaml:"example,omitempty"`

	ruleCache map[string]*BreakingChangeRule
	cacheOnce sync.Once
}

// Merge applies user overrides to the configuration. Only non-nil values from
// the override config replace the current values. Uses reflection to reduce boilerplate.
func (c *BreakingRulesConfig) Merge(override *BreakingRulesConfig) {
	if override == nil {
		return
	}

	cVal := reflect.ValueOf(c).Elem()
	oVal := reflect.ValueOf(override).Elem()

	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)
		cField := cVal.Field(i)
		oField := oVal.Field(i)

		if !cField.CanSet() {
			continue
		}

		if field.Type == ruleType {
			cField.Set(reflect.ValueOf(mergeRule(
				cField.Interface().(*BreakingChangeRule),
				oField.Interface().(*BreakingChangeRule),
			)))
			continue
		}

		if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
			if oField.IsNil() {
				continue
			}
			if cField.IsNil() {
				cField.Set(reflect.New(field.Type.Elem()))
			}
			mergeRulesStruct(cField.Elem(), oField.Elem())
		}
	}

	c.invalidateCache()
}

// IsBreaking looks up whether a change is breaking based on the component, property, and change type.
// Returns the configured breaking status, or false if the rule is not found.
func (c *BreakingRulesConfig) IsBreaking(component, property, changeType string) bool {
	rule := c.GetRule(component, property)
	if rule == nil {
		return false
	}

	switch changeType {
	case ChangeTypeAdded:
		if rule.Added != nil {
			return *rule.Added
		}
	case ChangeTypeModified:
		if rule.Modified != nil {
			return *rule.Modified
		}
	case ChangeTypeRemoved:
		if rule.Removed != nil {
			return *rule.Removed
		}
	}
	return false
}

// GetRule returns the BreakingChangeRule for a given component and property.
// Returns nil if no rule is defined. Uses internal cache for O(1) lookups.
func (c *BreakingRulesConfig) GetRule(component, property string) *BreakingChangeRule {
	c.cacheOnce.Do(func() {
		c.ruleCache = c.buildRuleCache()
	})
	if property == "" {
		return c.ruleCache[component]
	}
	return c.ruleCache[component+"."+property]
}

// buildRuleCache creates a flat map of all rules for O(1) lookups using reflection.
func (c *BreakingRulesConfig) buildRuleCache() map[string]*BreakingChangeRule {
	cache := make(map[string]*BreakingChangeRule, 200)
	cVal := reflect.ValueOf(c).Elem()

	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)
		fVal := cVal.Field(i)

		compName := jsonTagName(field)
		if compName == "" || !fVal.CanInterface() {
			continue
		}

		if field.Type == ruleType {
			cache[compName] = fVal.Interface().(*BreakingChangeRule)
			continue
		}

		if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
			if fVal.IsNil() {
				continue
			}
			addRulesToCache(cache, compName, fVal.Elem())
		}
	}
	return cache
}

// invalidateCache resets the cache so it will be rebuilt on next access.
func (c *BreakingRulesConfig) invalidateCache() {
	c.cacheOnce = sync.Once{}
	c.ruleCache = nil
}

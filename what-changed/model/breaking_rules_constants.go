// Copyright 2022-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"reflect"
	"sync"
)

// Component name constants for breaking change rule lookups.
// These match the JSON keys used in BreakingRulesConfig.
const (
	CompCallback            = "callback"
	CompContact             = "contact"
	CompDiscriminator       = "discriminator"
	CompEncoding            = "encoding"
	CompExample             = "example"
	CompExternalDocs        = "externalDocs"
	CompHeader              = "header"
	CompInfo                = "info"
	CompLicense             = "license"
	CompLink                = "link"
	CompMediaType           = "mediaType"
	CompOAuthFlow           = "oauthFlow"
	CompOAuthFlows          = "oauthFlows"
	CompOperation           = "operation"
	CompParameter           = "parameter"
	CompPathItem            = "pathItem"
	CompPaths               = "paths"
	CompRequestBody         = "requestBody"
	CompResponse            = "response"
	CompResponses           = "responses"
	CompSchema              = "schema"
	CompSecurityRequirement = "securityRequirement"
	CompSecurityScheme      = "securityScheme"
	CompServer              = "server"
	CompServerVariable      = "serverVariable"
	CompTag                 = "tag"
	CompXML                 = "xml"
)

// Property name constants for breaking change rule lookups.
// These match the JSON keys used in the various *Rules structs.
const (
	PropAdditionalOperations = "additionalOperations"
	PropAdditionalProperties = "additionalProperties"
	PropAllOf                = "allOf"
	PropAllowEmptyValue      = "allowEmptyValue"
	PropAllowReserved        = "allowReserved"
	PropAnyOf                = "anyOf"
	PropAttribute            = "attribute"
	PropAuthorizationCode    = "authorizationCode"
	PropAuthorizationURL     = "authorizationUrl"
	PropBearerFormat         = "bearerFormat"
	PropCallbacks            = "callbacks"
	PropClientCredentials    = "clientCredentials"
	PropConst                = "const"
	PropContact              = "contact"
	PropContains             = "contains"
	PropContentEncoding      = "contentEncoding"
	PropContentMediaType     = "contentMediaType"
	PropContentType          = "contentType"
	PropDataValue            = "dataValue"
	PropDefault              = "default"
	PropDefaultMapping       = "defaultMapping"
	PropDelete               = "delete"
	PropDeprecated           = "deprecated"
	PropDependentRequired    = "dependentRequired"
	PropDescription          = "description"
	PropDevice               = "device"
	PropDiscriminator        = "discriminator"
	PropElse                 = "else"
	PropEmail                = "email"
	PropEnum                 = "enum"
	PropExample              = "example"
	PropExclusiveMaximum     = "exclusiveMaximum"
	PropExclusiveMinimum     = "exclusiveMinimum"
	PropExplode              = "explode"
	PropExpressions          = "expressions"
	PropExternalDocs         = "externalDocs"
	PropExternalValue        = "externalValue"
	PropFlows                = "flows"
	PropFormat               = "format"
	PropGet                  = "get"
	PropHead                 = "head"
	PropIdentifier           = "identifier"
	PropIf                   = "if"
	PropImplicit             = "implicit"
	PropIn                   = "in"
	PropItems                = "items"
	PropItemEncoding         = "itemEncoding"
	PropItemSchema           = "itemSchema"
	PropKind                 = "kind"
	PropLicense              = "license"
	PropMapping              = "mapping"
	PropMaxItems             = "maxItems"
	PropMaxLength            = "maxLength"
	PropMaxProperties        = "maxProperties"
	PropMaximum              = "maximum"
	PropMinItems             = "minItems"
	PropMinLength            = "minLength"
	PropMinProperties        = "minProperties"
	PropMinimum              = "minimum"
	PropMultipleOf           = "multipleOf"
	PropName                 = "name"
	PropNamespace            = "namespace"
	PropNodeType             = "nodeType"
	PropNot                  = "not"
	PropNullable             = "nullable"
	PropOAuth2MetadataUrl    = "oauth2MetadataUrl"
	PropOneOf                = "oneOf"
	PropOpenIDConnectURL     = "openIdConnectUrl"
	PropOperationID          = "operationId"
	PropOperationRef         = "operationRef"
	PropOptions              = "options"
	PropParameters           = "parameters"
	PropParent               = "parent"
	PropPassword             = "password"
	PropPatch                = "patch"
	PropPath                 = "path"
	PropPattern              = "pattern"
	PropPost                 = "post"
	PropPrefix               = "prefix"
	PropPrefixItems          = "prefixItems"
	PropProperties           = "properties"
	PropPropertyName         = "propertyName"
	PropPropertyNames        = "propertyNames"
	PropPut                  = "put"
	PropQuery                = "query"
	PropReadOnly             = "readOnly"
	PropRef                  = "$ref"
	PropRefreshURL           = "refreshUrl"
	PropRequired             = "required"
	PropRequestBody          = "requestBody"
	PropResponses            = "responses"
	PropScheme               = "scheme"
	PropSchemes              = "schemes"
	PropSchema               = "schema"
	PropScopes               = "scopes"
	PropSecurity             = "security"
	PropSelf                 = "$self"
	PropSerializedValue      = "serializedValue"
	PropServer               = "server"
	PropServers              = "servers"
	PropStyle                = "style"
	PropSummary              = "summary"
	PropTags                 = "tags"
	PropTermsOfService       = "termsOfService"
	PropThen                 = "then"
	PropTitle                = "title"
	PropTokenURL             = "tokenUrl"
	PropTrace                = "trace"
	PropType                 = "type"
	PropUnevaluatedItems     = "unevaluatedItems"
	PropUnevaluatedProps     = "unevaluatedProperties"
	PropUniqueItems          = "uniqueItems"
	PropURL                  = "url"
	PropValue                = "value"
	PropVersion              = "version"
	PropWrapped              = "wrapped"
	PropWriteOnly            = "writeOnly"
)

// ChangeType constants for IsBreaking lookup
const (
	ChangeTypeAdded    = "added"
	ChangeTypeModified = "modified"
	ChangeTypeRemoved  = "removed"
)

// reflection types cached at init to avoid repeated TypeOf calls in hot paths
var (
	ruleType   = reflect.TypeOf((*BreakingChangeRule)(nil))
	configType = reflect.TypeOf(BreakingRulesConfig{})
)

// singleton cache for default rules to avoid repeated allocations
var (
	defaultRulesOnce  sync.Once
	defaultRulesCache *BreakingRulesConfig
)

// active config used by comparison functions
var (
	activeConfigMu sync.RWMutex
	activeConfig   *BreakingRulesConfig
)

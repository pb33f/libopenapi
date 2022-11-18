// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

type SharedParameters interface {
	HasDescription
	Hash() [32]byte
	GetName() *NodeReference[string]
	GetIn() *NodeReference[string]
	GetAllowEmptyValue() *NodeReference[bool]
	GetRequired() *NodeReference[bool]
	GetSchema() *NodeReference[any] // requires cast.
}

type HasExternalDocs interface {
	GetExternalDocs() *NodeReference[any]
}

type HasDescription interface {
	GetDescription() *NodeReference[string]
}

type HasInfo interface {
	GetInfo() *NodeReference[any]
}

type SwaggerParameter interface {
	SharedParameters
	GetType() *NodeReference[string]
	GetFormat() *NodeReference[string]
	GetCollectionFormat() *NodeReference[string]
	GetDefault() *NodeReference[any]
	GetMaximum() *NodeReference[int]
	GetExclusiveMaximum() *NodeReference[bool]
	GetMinimum() *NodeReference[int]
	GetExclusiveMinimum() *NodeReference[bool]
	GetMaxLength() *NodeReference[int]
	GetMinLength() *NodeReference[int]
	GetPattern() *NodeReference[string]
	GetMaxItems() *NodeReference[int]
	GetMinItems() *NodeReference[int]
	GetUniqueItems() *NodeReference[bool]
	GetEnum() *NodeReference[[]ValueReference[any]]
	GetMultipleOf() *NodeReference[int]
}

type SwaggerHeader interface {
	HasDescription
	Hash() [32]byte
	GetType() *NodeReference[string]
	GetFormat() *NodeReference[string]
	GetCollectionFormat() *NodeReference[string]
	GetDefault() *NodeReference[any]
	GetMaximum() *NodeReference[int]
	GetExclusiveMaximum() *NodeReference[bool]
	GetMinimum() *NodeReference[int]
	GetExclusiveMinimum() *NodeReference[bool]
	GetMaxLength() *NodeReference[int]
	GetMinLength() *NodeReference[int]
	GetPattern() *NodeReference[string]
	GetMaxItems() *NodeReference[int]
	GetMinItems() *NodeReference[int]
	GetUniqueItems() *NodeReference[bool]
	GetEnum() *NodeReference[[]ValueReference[any]]
	GetMultipleOf() *NodeReference[int]
	GetItems() *NodeReference[any] // requires cast.
}

type OpenAPIHeader interface {
	HasDescription
	Hash() [32]byte
	GetDeprecated() *NodeReference[bool]
	GetStyle() *NodeReference[string]
	GetAllowReserved() *NodeReference[bool]
	GetExplode() *NodeReference[bool]
	GetExample() *NodeReference[any]
	GetRequired() *NodeReference[bool]
	GetAllowEmptyValue() *NodeReference[bool]
	GetSchema() *NodeReference[any]   // requires cast.
	GetExamples() *NodeReference[any] // requires cast.
	GetContent() *NodeReference[any]  // requires cast.
}

type OpenAPIParameter interface {
	SharedParameters
	GetDeprecated() *NodeReference[bool]
	GetStyle() *NodeReference[string]
	GetAllowReserved() *NodeReference[bool]
	GetExplode() *NodeReference[bool]
	GetExample() *NodeReference[any]
	GetExamples() *NodeReference[any] // requires cast.
	GetContent() *NodeReference[any]  // requires cast.
}

//TODO: this needs to be fixed, move returns to pointers.

type SharedOperations interface {
	//HasDescription
	//HasExternalDocs
	GetOperationId() NodeReference[string]
	GetExternalDocs() NodeReference[any]
	GetDescription() NodeReference[string]
	GetTags() NodeReference[[]ValueReference[string]]
	GetSummary() NodeReference[string]
	GetDeprecated() NodeReference[bool]
	GetExtensions() map[KeyReference[string]]ValueReference[any]
	GetResponses() NodeReference[any]  // requires cast.
	GetParameters() NodeReference[any] // requires cast.
	GetSecurity() NodeReference[any]   // requires cast.
}

type SwaggerOperations interface {
	SharedOperations
	GetConsumes() NodeReference[[]ValueReference[string]]
	GetProduces() NodeReference[[]ValueReference[string]]
	GetSchemes() NodeReference[[]ValueReference[string]]
}

type OpenAPIOperations interface {
	SharedOperations
	GetCallbacks() NodeReference[map[KeyReference[string]]ValueReference[any]] // requires cast
	GetServers() NodeReference[any]                                            // requires cast.
}

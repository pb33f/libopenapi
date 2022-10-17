// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

type IsParameter interface {
	GetName() *NodeReference[string]
	GetIn() *NodeReference[string]
	GetType() *NodeReference[string]
	GetDescription() *NodeReference[string]
	GetRequired() *NodeReference[bool]
	GetDeprecated() *NodeReference[bool]
	GetAllowEmptyValue() *NodeReference[bool]
	GetFormat() *NodeReference[string]
	GetStyle() *NodeReference[string]
	GetCollectionFormat() *NodeReference[string]
	GetDefault() *NodeReference[any]
	GetAllowReserved() *NodeReference[bool]
	GetExplode() *NodeReference[bool]
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
	GetEnum() *NodeReference[[]ValueReference[string]]
	GetMultipleOf() *NodeReference[int]
	GetExample() *NodeReference[any]
	GetExamples() *NodeReference[any] // requires cast
	GetSchema() *NodeReference[any]   // requires cast.
	GetContent() *NodeReference[any]  // requires cast.
	GetItems() *NodeReference[any]    // requires cast.
}

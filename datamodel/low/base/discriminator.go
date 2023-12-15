// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"crypto/sha256"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
)

// Discriminator is only used by OpenAPI 3+ documents, it represents a polymorphic discriminator used for schemas
//
// When request bodies or response payloads may be one of a number of different schemas, a discriminator object can be
// used to aid in serialization, deserialization, and validation. The discriminator is a specific object in a schema
// which is used to inform the consumer of the document of an alternative schema based on the value associated with it.
//
// When using the discriminator, inline schemas will not be considered.
//
//	v3 - https://spec.openapis.org/oas/v3.1.0#discriminator-object
type Discriminator struct {
	PropertyName low.NodeReference[string]
	Mapping      low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[string]]]
	low.Reference
}

// FindMappingValue will return a ValueReference containing the string mapping value
func (d *Discriminator) FindMappingValue(key string) *low.ValueReference[string] {
	for pair := orderedmap.First(d.Mapping.Value); pair != nil; pair = pair.Next() {
		if pair.Key().Value == key {
			v := pair.Value()
			return &v
		}
	}
	return nil
}

// Hash will return a consistent SHA256 Hash of the Discriminator object
func (d *Discriminator) Hash() [32]byte {
	// calculate a hash from every property.
	var f []string
	if d.PropertyName.Value != "" {
		f = append(f, d.PropertyName.Value)
	}

	for pair := orderedmap.First(orderedmap.SortAlpha(d.Mapping.Value)); pair != nil; pair = pair.Next() {
		f = append(f, pair.Value().Value)
	}

	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

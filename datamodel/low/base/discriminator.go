// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"crypto/sha256"
	"github.com/pb33f/libopenapi/datamodel/low"
	"sort"
	"strings"
)

// Discriminator is only used by OpenAPI 3+ documents, it represents a polymorphic discriminator used for schemas
//
// When request bodies or response payloads may be one of a number of different schemas, a discriminator object can be
// used to aid in serialization, deserialization, and validation. The discriminator is a specific object in a schema
// which is used to inform the consumer of the document of an alternative schema based on the value associated with it.
//
// When using the discriminator, inline schemas will not be considered.
//  v3 - https://spec.openapis.org/oas/v3.1.0#discriminator-object
type Discriminator struct {
	PropertyName low.NodeReference[string]
	Mapping      low.NodeReference[map[low.KeyReference[string]]low.ValueReference[string]]
	low.Reference
}

// FindMappingValue will return a ValueReference containing the string mapping value
func (d *Discriminator) FindMappingValue(key string) *low.ValueReference[string] {
	for k, v := range d.Mapping.Value {
		if k.Value == key {
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
	propertyKeys := make([]string, 0, len(d.Mapping.Value))
	for i := range d.Mapping.Value {
		propertyKeys = append(propertyKeys, i.Value)
	}
	sort.Strings(propertyKeys)
	for k := range propertyKeys {
		prop := d.FindMappingValue(propertyKeys[k])
		f = append(f, prop.Value)
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"sort"
	"sync"

	"slices"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

// SchemaChanges represent all changes to a base.Schema OpenAPI object. These changes are represented
// by all versions of OpenAPI.
//
// Any additions or removals to slice based results will be recorded in the PropertyChanges of the parent
// changes, and not the child for example, adding a new schema to `anyOf` will create a new change result in
// PropertyChanges.Changes, and not in the AnyOfChanges property.
type SchemaChanges struct {
	*PropertyChanges
	DiscriminatorChanges        *DiscriminatorChanges     `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`
	AllOfChanges                []*SchemaChanges          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	AnyOfChanges                []*SchemaChanges          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	OneOfChanges                []*SchemaChanges          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	NotChanges                  *SchemaChanges            `json:"not,omitempty" yaml:"not,omitempty"`
	ItemsChanges                *SchemaChanges            `json:"items,omitempty" yaml:"items,omitempty"`
	SchemaPropertyChanges       map[string]*SchemaChanges `json:"properties,omitempty" yaml:"properties,omitempty"`
	ExternalDocChanges          *ExternalDocChanges       `json:"externalDoc,omitempty" yaml:"externalDoc,omitempty"`
	XMLChanges                  *XMLChanges               `json:"xml,omitempty" yaml:"xml,omitempty"`
	ExtensionChanges            *ExtensionChanges         `json:"extensions,omitempty" yaml:"extensions,omitempty"`
	AdditionalPropertiesChanges *SchemaChanges            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`

	// 3.1 specifics
	IfChanges                    *SchemaChanges            `json:"if,omitempty" yaml:"if,omitempty"`
	ElseChanges                  *SchemaChanges            `json:"else,omitempty" yaml:"else,omitempty"`
	ThenChanges                  *SchemaChanges            `json:"then,omitempty" yaml:"then,omitempty"`
	PropertyNamesChanges         *SchemaChanges            `json:"propertyNames,omitempty" yaml:"propertyNames,omitempty"`
	ContainsChanges              *SchemaChanges            `json:"contains,omitempty" yaml:"contains,omitempty"`
	UnevaluatedItemsChanges      *SchemaChanges            `json:"unevaluatedItems,omitempty" yaml:"unevaluatedItems,omitempty"`
	UnevaluatedPropertiesChanges *SchemaChanges            `json:"unevaluatedProperties,omitempty" yaml:"unevaluatedProperties,omitempty"`
	DependentSchemasChanges      map[string]*SchemaChanges `json:"dependentSchemas,omitempty" yaml:"dependentSchemas,omitempty"`
	PatternPropertiesChanges     map[string]*SchemaChanges `json:"patternProperties,omitempty" yaml:"patternProperties,omitempty"`
}

// GetAllChanges returns a slice of all changes made between Responses objects
func (s *SchemaChanges) GetAllChanges() []*Change {
	var changes []*Change
	changes = append(changes, s.Changes...)
	if s.DiscriminatorChanges != nil {
		changes = append(changes, s.DiscriminatorChanges.GetAllChanges()...)
	}
	if len(s.AllOfChanges) > 0 {
		for n := range s.AllOfChanges {
			if s.AllOfChanges[n] != nil {
				changes = append(changes, s.AllOfChanges[n].GetAllChanges()...)
			}
		}
	}
	if len(s.AnyOfChanges) > 0 {
		for n := range s.AnyOfChanges {
			if s.AnyOfChanges[n] != nil {
				changes = append(changes, s.AnyOfChanges[n].GetAllChanges()...)
			}
		}
	}
	if len(s.OneOfChanges) > 0 {
		for n := range s.OneOfChanges {
			if s.OneOfChanges[n] != nil {
				changes = append(changes, s.OneOfChanges[n].GetAllChanges()...)
			}
		}
	}
	if s.NotChanges != nil {
		changes = append(changes, s.NotChanges.GetAllChanges()...)
	}
	if s.ItemsChanges != nil {
		changes = append(changes, s.ItemsChanges.GetAllChanges()...)
	}
	if s.IfChanges != nil {
		changes = append(changes, s.IfChanges.GetAllChanges()...)
	}
	if s.ElseChanges != nil {
		changes = append(changes, s.ElseChanges.GetAllChanges()...)
	}
	if s.ThenChanges != nil {
		changes = append(changes, s.ThenChanges.GetAllChanges()...)
	}
	if s.PropertyNamesChanges != nil {
		changes = append(changes, s.PropertyNamesChanges.GetAllChanges()...)
	}
	if s.ContainsChanges != nil {
		changes = append(changes, s.ContainsChanges.GetAllChanges()...)
	}
	if s.UnevaluatedItemsChanges != nil {
		changes = append(changes, s.UnevaluatedItemsChanges.GetAllChanges()...)
	}
	if s.UnevaluatedPropertiesChanges != nil {
		changes = append(changes, s.UnevaluatedPropertiesChanges.GetAllChanges()...)
	}
	if s.AdditionalPropertiesChanges != nil {
		changes = append(changes, s.AdditionalPropertiesChanges.GetAllChanges()...)
	}
	if s.SchemaPropertyChanges != nil {
		for n := range s.SchemaPropertyChanges {
			if s.SchemaPropertyChanges[n] != nil {
				changes = append(changes, s.SchemaPropertyChanges[n].GetAllChanges()...)
			}
		}
	}
	if s.DependentSchemasChanges != nil {
		for n := range s.DependentSchemasChanges {
			if s.DependentSchemasChanges[n] != nil {
				changes = append(changes, s.DependentSchemasChanges[n].GetAllChanges()...)
			}
		}
	}
	if s.PatternPropertiesChanges != nil {
		for n := range s.PatternPropertiesChanges {
			if s.PatternPropertiesChanges[n] != nil {
				changes = append(changes, s.PatternPropertiesChanges[n].GetAllChanges()...)
			}
		}
	}
	if s.ExternalDocChanges != nil {
		changes = append(changes, s.ExternalDocChanges.GetAllChanges()...)
	}
	if s.XMLChanges != nil {
		changes = append(changes, s.XMLChanges.GetAllChanges()...)
	}
	if s.ExtensionChanges != nil {
		changes = append(changes, s.ExtensionChanges.GetAllChanges()...)
	}
	return changes
}

// TotalChanges returns a count of the total number of changes made to this schema and all sub-schemas
func (s *SchemaChanges) TotalChanges() int {
	if s == nil {
		return 0
	}

	t := s.PropertyChanges.TotalChanges()
	if s.DiscriminatorChanges != nil {
		t += s.DiscriminatorChanges.TotalChanges()
	}
	if len(s.AllOfChanges) > 0 {
		for n := range s.AllOfChanges {
			t += s.AllOfChanges[n].TotalChanges()
		}
	}
	if len(s.AnyOfChanges) > 0 {
		for n := range s.AnyOfChanges {
			if s.AnyOfChanges[n] != nil {
				t += s.AnyOfChanges[n].TotalChanges()
			}
		}
	}
	if len(s.OneOfChanges) > 0 {
		for n := range s.OneOfChanges {
			t += s.OneOfChanges[n].TotalChanges()
		}
	}
	if s.NotChanges != nil {
		t += s.NotChanges.TotalChanges()
	}
	if s.ItemsChanges != nil {
		t += s.ItemsChanges.TotalChanges()
	}
	if s.IfChanges != nil {
		t += s.IfChanges.TotalChanges()
	}
	if s.ElseChanges != nil {
		t += s.ElseChanges.TotalChanges()
	}
	if s.ThenChanges != nil {
		t += s.ThenChanges.TotalChanges()
	}
	if s.PropertyNamesChanges != nil {
		t += s.PropertyNamesChanges.TotalChanges()
	}
	if s.ContainsChanges != nil {
		t += s.ContainsChanges.TotalChanges()
	}
	if s.UnevaluatedItemsChanges != nil {
		t += s.UnevaluatedItemsChanges.TotalChanges()
	}
	if s.UnevaluatedPropertiesChanges != nil {
		t += s.UnevaluatedPropertiesChanges.TotalChanges()
	}
	if s.AdditionalPropertiesChanges != nil {
		t += s.AdditionalPropertiesChanges.TotalChanges()
	}
	if s.SchemaPropertyChanges != nil {
		for n := range s.SchemaPropertyChanges {
			if s.SchemaPropertyChanges[n] != nil {
				t += s.SchemaPropertyChanges[n].TotalChanges()
			}
		}
	}
	if s.DependentSchemasChanges != nil {
		for n := range s.DependentSchemasChanges {
			t += s.DependentSchemasChanges[n].TotalChanges()
		}
	}
	if s.PatternPropertiesChanges != nil {
		for n := range s.PatternPropertiesChanges {
			t += s.PatternPropertiesChanges[n].TotalChanges()
		}
	}
	if s.ExternalDocChanges != nil {
		t += s.ExternalDocChanges.TotalChanges()
	}
	if s.XMLChanges != nil {
		t += s.XMLChanges.TotalChanges()
	}
	if s.ExtensionChanges != nil {
		t += s.ExtensionChanges.TotalChanges()
	}
	return t
}

// TotalBreakingChanges returns the total number of breaking changes made to this schema and all sub-schemas.
func (s *SchemaChanges) TotalBreakingChanges() int {
	if s == nil {
		return 0
	}

	t := s.PropertyChanges.TotalBreakingChanges()
	if s.DiscriminatorChanges != nil {
		t += s.DiscriminatorChanges.TotalBreakingChanges()
	}
	if len(s.AllOfChanges) > 0 {
		for n := range s.AllOfChanges {
			t += s.AllOfChanges[n].TotalBreakingChanges()
		}
	}
	if len(s.AllOfChanges) > 0 {
		for n := range s.AllOfChanges {
			t += s.AllOfChanges[n].TotalBreakingChanges()
		}
	}
	if len(s.AnyOfChanges) > 0 {
		for n := range s.AnyOfChanges {
			t += s.AnyOfChanges[n].TotalBreakingChanges()
		}
	}
	if len(s.OneOfChanges) > 0 {
		for n := range s.OneOfChanges {
			t += s.OneOfChanges[n].TotalBreakingChanges()
		}
	}
	if s.NotChanges != nil {
		t += s.NotChanges.TotalBreakingChanges()
	}
	if s.ItemsChanges != nil {
		t += s.ItemsChanges.TotalBreakingChanges()
	}
	if s.IfChanges != nil {
		t += s.IfChanges.TotalBreakingChanges()
	}
	if s.ElseChanges != nil {
		t += s.ElseChanges.TotalBreakingChanges()
	}
	if s.ThenChanges != nil {
		t += s.ThenChanges.TotalBreakingChanges()
	}
	if s.PropertyNamesChanges != nil {
		t += s.PropertyNamesChanges.TotalBreakingChanges()
	}
	if s.ContainsChanges != nil {
		t += s.ContainsChanges.TotalBreakingChanges()
	}
	if s.UnevaluatedItemsChanges != nil {
		t += s.UnevaluatedItemsChanges.TotalBreakingChanges()
	}
	if s.UnevaluatedPropertiesChanges != nil {
		t += s.UnevaluatedPropertiesChanges.TotalBreakingChanges()
	}
	if s.AdditionalPropertiesChanges != nil {
		t += s.AdditionalPropertiesChanges.TotalBreakingChanges()
	}
	if s.DependentSchemasChanges != nil {
		for n := range s.DependentSchemasChanges {
			t += s.DependentSchemasChanges[n].TotalBreakingChanges()
		}
	}
	if s.PatternPropertiesChanges != nil {
		for n := range s.PatternPropertiesChanges {
			t += s.PatternPropertiesChanges[n].TotalBreakingChanges()
		}
	}
	if s.XMLChanges != nil {
		t += s.XMLChanges.TotalBreakingChanges()
	}
	if s.SchemaPropertyChanges != nil {
		for n := range s.SchemaPropertyChanges {
			t += s.SchemaPropertyChanges[n].TotalBreakingChanges()
		}
	}
	return t
}

// CompareSchemas accepts a left and right SchemaProxy and checks for changes. If anything is found, returns
// a pointer to SchemaChanges, otherwise returns nil
func CompareSchemas(l, r *base.SchemaProxy) *SchemaChanges {
	sc := new(SchemaChanges)
	var changes []*Change

	// Added
	if l == nil && r != nil {
		CreateChange(&changes, ObjectAdded, v3.SchemaLabel,
			nil, nil, true, nil, r)
		sc.PropertyChanges = NewPropertyChanges(changes)
	}

	// Removed
	if l != nil && r == nil {
		CreateChange(&changes, ObjectRemoved, v3.SchemaLabel,
			nil, nil, true, l, nil)
		sc.PropertyChanges = NewPropertyChanges(changes)
	}

	if l != nil && r != nil {

		// if left proxy is a reference and right is a reference (we won't recurse into them)
		if l.IsReference() && r.IsReference() {
			// points to the same schema
			if l.GetReference() == r.GetReference() {
				// there is nothing to be done at this point.
				return nil
			} else {
				// references are different, that's all we care to know.
				CreateChange(&changes, Modified, v3.RefLabel,
					l.GetValueNode().Content[1], r.GetValueNode().Content[1], true, l.GetReference(),
					r.GetReference())
				sc.PropertyChanges = NewPropertyChanges(changes)
				return sc
			}
		}

		// changed from inline to ref
		if !l.IsReference() && r.IsReference() {
			// check if the referenced schema matches or not
			// https://github.com/pb33f/libopenapi/issues/218
			lHash := l.Schema().Hash()
			rHash := r.Schema().Hash()
			if lHash != rHash {
				CreateChange(&changes, Modified, v3.RefLabel,
					l.GetValueNode(), r.GetValueNode().Content[1], true, l, r.GetReference())
				sc.PropertyChanges = NewPropertyChanges(changes)
				return sc // we're done here
			}
		}

		// changed from ref to inline
		if l.IsReference() && !r.IsReference() {
			// check if the referenced schema matches or not
			// https://github.com/pb33f/libopenapi/issues/218
			lHash := l.Schema().Hash()
			rHash := r.Schema().Hash()
			if lHash != rHash {
				CreateChange(&changes, Modified, v3.RefLabel,
					l.GetValueNode().Content[1], r.GetValueNode(), true, l.GetReference(), r)
				sc.PropertyChanges = NewPropertyChanges(changes)
				return sc // done, nothing else to do.
			}
		}

		lSchema := l.Schema()
		rSchema := r.Schema()

		if low.AreEqual(lSchema, rSchema) {
			// there is no point going on, we know nothing changed!
			return nil
		}

		// check XML
		checkSchemaXML(lSchema, rSchema, &changes, sc)

		// check examples
		checkExamples(lSchema, rSchema, &changes)

		// check schema core properties for changes.
		checkSchemaPropertyChanges(lSchema, rSchema, &changes, sc)

		// now for the confusing part, there is also a schema's 'properties' property to parse.
		// inception, eat your heart out.
		doneChan := make(chan bool)
		props, totalProperties := checkMappedSchemaOfASchema(lSchema.Properties.Value, rSchema.Properties.Value, &changes, doneChan)
		sc.SchemaPropertyChanges = props

		deps, depsTotal := checkMappedSchemaOfASchema(lSchema.DependentSchemas.Value, rSchema.DependentSchemas.Value, &changes, doneChan)
		sc.DependentSchemasChanges = deps

		patterns, patternsTotal := checkMappedSchemaOfASchema(lSchema.PatternProperties.Value, rSchema.PatternProperties.Value, &changes, doneChan)
		sc.PatternPropertiesChanges = patterns

		// check polymorphic and multi-values async for speed.
		go extractSchemaChanges(lSchema.OneOf.Value, rSchema.OneOf.Value, v3.OneOfLabel,
			&sc.OneOfChanges, &changes, doneChan)

		go extractSchemaChanges(lSchema.AllOf.Value, rSchema.AllOf.Value, v3.AllOfLabel,
			&sc.AllOfChanges, &changes, doneChan)

		go extractSchemaChanges(lSchema.AnyOf.Value, rSchema.AnyOf.Value, v3.AnyOfLabel,
			&sc.AnyOfChanges, &changes, doneChan)

		totalChecks := totalProperties + depsTotal + patternsTotal + 3
		completedChecks := 0
		for completedChecks < totalChecks {
			<-doneChan
			completedChecks++
		}
	}
	// done
	if changes != nil {
		sc.PropertyChanges = NewPropertyChanges(changes)
	} else {
		sc.PropertyChanges = NewPropertyChanges(nil)
	}
	if sc.TotalChanges() > 0 {
		return sc
	}
	return nil
}

func checkSchemaXML(lSchema *base.Schema, rSchema *base.Schema, changes *[]*Change, sc *SchemaChanges) {
	// XML removed
	if lSchema.XML.Value != nil && rSchema.XML.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.XMLLabel,
			lSchema.XML.GetValueNode(), nil, true, lSchema.XML.GetValue(), nil)
	}
	// XML added
	if lSchema.XML.Value == nil && rSchema.XML.Value != nil {
		CreateChange(changes, ObjectAdded, v3.XMLLabel,
			nil, rSchema.XML.GetValueNode(), false, nil, rSchema.XML.GetValue())
	}

	// compare XML
	if lSchema.XML.Value != nil && rSchema.XML.Value != nil {
		if !low.AreEqual(lSchema.XML.Value, rSchema.XML.Value) {
			sc.XMLChanges = CompareXML(lSchema.XML.Value, rSchema.XML.Value)
		}
	}
}

func checkMappedSchemaOfASchema(
	lSchema,
	rSchema *orderedmap.Map[low.KeyReference[string], low.ValueReference[*base.SchemaProxy]],
	changes *[]*Change,
	doneChan chan bool,
) (map[string]*SchemaChanges, int) {
	propChanges := make(map[string]*SchemaChanges)

	var lProps []string
	lEntities := make(map[string]*base.SchemaProxy)
	lKeyNodes := make(map[string]*yaml.Node)
	var rProps []string
	rEntities := make(map[string]*base.SchemaProxy)
	rKeyNodes := make(map[string]*yaml.Node)

	for pair := orderedmap.First(lSchema); pair != nil; pair = pair.Next() {
		lProps = append(lProps, pair.Key().Value)
		lEntities[pair.Key().Value] = pair.Value().Value
		lKeyNodes[pair.Key().Value] = pair.Key().KeyNode
	}
	for pair := orderedmap.First(rSchema); pair != nil; pair = pair.Next() {
		rProps = append(rProps, pair.Key().Value)
		rEntities[pair.Key().Value] = pair.Value().Value
		rKeyNodes[pair.Key().Value] = pair.Key().KeyNode
	}
	sort.Strings(lProps)
	sort.Strings(rProps)
	totalProperties := buildProperty(lProps, rProps, lEntities, rEntities, propChanges, doneChan, changes, rKeyNodes, lKeyNodes)
	return propChanges, totalProperties
}

func buildProperty(lProps, rProps []string, lEntities, rEntities map[string]*base.SchemaProxy,
	propChanges map[string]*SchemaChanges, doneChan chan bool, changes *[]*Change, rKeyNodes, lKeyNodes map[string]*yaml.Node,
) int {
	var propLock sync.Mutex
	checkProperty := func(key string, lp, rp *base.SchemaProxy, propChanges map[string]*SchemaChanges, done chan bool) {
		if lp != nil && rp != nil {
			if low.AreEqual(lp, rp) {
				done <- true
				return
			}
			s := CompareSchemas(lp, rp)
			propLock.Lock()
			propChanges[key] = s
			propLock.Unlock()
			done <- true
		}
	}

	totalProperties := 0
	if len(lProps) == len(rProps) {
		for w := range lProps {
			lp := lEntities[lProps[w]]
			rp := rEntities[rProps[w]]
			if lProps[w] == rProps[w] && lp != nil && rp != nil {
				totalProperties++
				go checkProperty(lProps[w], lp, rp, propChanges, doneChan)
			}

			// keys do not match, even after sorting, check which one was added and which one was removed.
			if lProps[w] != rProps[w] {
				if !slices.Contains(lProps, rProps[w]) {
					// new added.
					CreateChange(changes, ObjectAdded, v3.PropertiesLabel,
						nil, rKeyNodes[rProps[w]], false, nil, rEntities[rProps[w]])
				}
				if !slices.Contains(rProps, lProps[w]) {
					CreateChange(changes, ObjectRemoved, v3.PropertiesLabel,
						lKeyNodes[lProps[w]], nil, true, lEntities[lProps[w]], nil)
				}
				if slices.Contains(lProps, rProps[w]) {
					h := slices.Index(lProps, rProps[w])
					totalProperties++
					lp = lEntities[lProps[h]]
					rp = rEntities[rProps[w]]
					go checkProperty(lProps[h], lp, rp, propChanges, doneChan)
				}
			}
		}
	}

	// something removed
	if len(lProps) > len(rProps) {
		for w := range lProps {
			if rEntities[lProps[w]] != nil {
				totalProperties++
				go checkProperty(lProps[w], lEntities[lProps[w]], rEntities[lProps[w]], propChanges, doneChan)
				continue
			} else {
				CreateChange(changes, ObjectRemoved, v3.PropertiesLabel,
					lKeyNodes[lProps[w]], nil, true, lEntities[lProps[w]], nil)
				continue
			}
		}
	}

	// something added
	if len(rProps) > len(lProps) {
		for w := range rProps {
			if lEntities[rProps[w]] != nil {
				totalProperties++
				go checkProperty(rProps[w], lEntities[rProps[w]], rEntities[rProps[w]], propChanges, doneChan)
			} else {
				CreateChange(changes, ObjectAdded, v3.PropertiesLabel,
					nil, rKeyNodes[rProps[w]], false, nil, rEntities[rProps[w]])
				continue
			}
		}
	}
	return totalProperties
}

func checkSchemaPropertyChanges(
	lSchema *base.Schema,
	rSchema *base.Schema,
	changes *[]*Change, sc *SchemaChanges,
) {
	var props []*PropertyCheck

	// $schema (breaking change)
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.SchemaTypeRef.ValueNode,
		RightNode: rSchema.SchemaTypeRef.ValueNode,
		Label:     v3.SchemaDialectLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// ExclusiveMaximum
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.ExclusiveMaximum.ValueNode,
		RightNode: rSchema.ExclusiveMaximum.ValueNode,
		Label:     v3.ExclusiveMaximumLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// ExclusiveMinimum
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.ExclusiveMinimum.ValueNode,
		RightNode: rSchema.ExclusiveMinimum.ValueNode,
		Label:     v3.ExclusiveMinimumLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// Type
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.Type.ValueNode,
		RightNode: rSchema.Type.ValueNode,
		Label:     v3.TypeLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// Title
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.Title.ValueNode,
		RightNode: rSchema.Title.ValueNode,
		Label:     v3.TitleLabel,
		Changes:   changes,
		Breaking:  false,
		Original:  lSchema,
		New:       rSchema,
	})

	// MultipleOf
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.MultipleOf.ValueNode,
		RightNode: rSchema.MultipleOf.ValueNode,
		Label:     v3.MultipleOfLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// Maximum
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.Maximum.ValueNode,
		RightNode: rSchema.Maximum.ValueNode,
		Label:     v3.MaximumLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// Minimum
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.Minimum.ValueNode,
		RightNode: rSchema.Minimum.ValueNode,
		Label:     v3.MinimumLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// MaxLength
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.MaxLength.ValueNode,
		RightNode: rSchema.MaxLength.ValueNode,
		Label:     v3.MaxLengthLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// MinLength
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.MinLength.ValueNode,
		RightNode: rSchema.MinLength.ValueNode,
		Label:     v3.MinLengthLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// Pattern
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.Pattern.ValueNode,
		RightNode: rSchema.Pattern.ValueNode,
		Label:     v3.PatternLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// Format
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.Format.ValueNode,
		RightNode: rSchema.Format.ValueNode,
		Label:     v3.FormatLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// MaxItems
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.MaxItems.ValueNode,
		RightNode: rSchema.MaxItems.ValueNode,
		Label:     v3.MaxItemsLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// MinItems
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.MinItems.ValueNode,
		RightNode: rSchema.MinItems.ValueNode,
		Label:     v3.MinItemsLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// MaxProperties
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.MaxProperties.ValueNode,
		RightNode: rSchema.MaxProperties.ValueNode,
		Label:     v3.MaxPropertiesLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// MinProperties
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.MinProperties.ValueNode,
		RightNode: rSchema.MinProperties.ValueNode,
		Label:     v3.MinPropertiesLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// UniqueItems
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.UniqueItems.ValueNode,
		RightNode: rSchema.UniqueItems.ValueNode,
		Label:     v3.UniqueItemsLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// AdditionalProperties
	if lSchema.AdditionalProperties.Value != nil && rSchema.AdditionalProperties.Value != nil {
		if lSchema.AdditionalProperties.Value.IsA() && rSchema.AdditionalProperties.Value.IsA() {
			if !low.AreEqual(lSchema.AdditionalProperties.Value.A, rSchema.AdditionalProperties.Value.A) {
				sc.AdditionalPropertiesChanges = CompareSchemas(lSchema.AdditionalProperties.Value.A, rSchema.AdditionalProperties.Value.A)
			}
		} else {
			if lSchema.AdditionalProperties.Value.IsB() && rSchema.AdditionalProperties.Value.IsB() {
				if lSchema.AdditionalProperties.Value.B != rSchema.AdditionalProperties.Value.B {
					CreateChange(changes, Modified, v3.AdditionalPropertiesLabel,
						lSchema.AdditionalProperties.ValueNode, rSchema.AdditionalProperties.ValueNode, true,
						lSchema.AdditionalProperties.Value.B, rSchema.AdditionalProperties.Value.B)
				}
			} else {
				CreateChange(changes, Modified, v3.AdditionalPropertiesLabel,
					lSchema.AdditionalProperties.ValueNode, rSchema.AdditionalProperties.ValueNode, true,
					lSchema.AdditionalProperties.Value.B, rSchema.AdditionalProperties.Value.B)
			}
		}
	}

	// added AdditionalProperties
	if lSchema.AdditionalProperties.Value == nil && rSchema.AdditionalProperties.Value != nil {
		CreateChange(changes, ObjectAdded, v3.AdditionalPropertiesLabel,
			nil, rSchema.AdditionalProperties.ValueNode, true, nil, rSchema.AdditionalProperties.Value)
	}
	// removed AdditionalProperties
	if lSchema.AdditionalProperties.Value != nil && rSchema.AdditionalProperties.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.AdditionalPropertiesLabel,
			lSchema.AdditionalProperties.ValueNode, nil, true, lSchema.AdditionalProperties.Value, nil)
	}

	// Description
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.Description.ValueNode,
		RightNode: rSchema.Description.ValueNode,
		Label:     v3.DescriptionLabel,
		Changes:   changes,
		Breaking:  false,
		Original:  lSchema,
		New:       rSchema,
	})

	// ContentEncoding
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.ContentEncoding.ValueNode,
		RightNode: rSchema.ContentEncoding.ValueNode,
		Label:     v3.ContentEncodingLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// ContentMediaType
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.ContentMediaType.ValueNode,
		RightNode: rSchema.ContentMediaType.ValueNode,
		Label:     v3.ContentMediaType,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// Default
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.Default.ValueNode,
		RightNode: rSchema.Default.ValueNode,
		Label:     v3.DefaultLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// Const
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.Const.ValueNode,
		RightNode: rSchema.Const.ValueNode,
		Label:     v3.ConstLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// Nullable
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.Nullable.ValueNode,
		RightNode: rSchema.Nullable.ValueNode,
		Label:     v3.NullableLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// ReadOnly
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.ReadOnly.ValueNode,
		RightNode: rSchema.ReadOnly.ValueNode,
		Label:     v3.ReadOnlyLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// WriteOnly
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.WriteOnly.ValueNode,
		RightNode: rSchema.WriteOnly.ValueNode,
		Label:     v3.WriteOnlyLabel,
		Changes:   changes,
		Breaking:  true,
		Original:  lSchema,
		New:       rSchema,
	})

	// Example
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.Example.ValueNode,
		RightNode: rSchema.Example.ValueNode,
		Label:     v3.ExampleLabel,
		Changes:   changes,
		Breaking:  false,
		Original:  lSchema,
		New:       rSchema,
	})

	// Deprecated
	props = append(props, &PropertyCheck{
		LeftNode:  lSchema.Deprecated.ValueNode,
		RightNode: rSchema.Deprecated.ValueNode,
		Label:     v3.DeprecatedLabel,
		Changes:   changes,
		Breaking:  false,
		Original:  lSchema,
		New:       rSchema,
	})

	// Required
	j := make(map[string]int)
	k := make(map[string]int)
	for i := range lSchema.Required.Value {
		j[lSchema.Required.Value[i].Value] = i
	}
	for i := range rSchema.Required.Value {
		k[rSchema.Required.Value[i].Value] = i
	}
	for g := range k {
		if _, ok := j[g]; !ok {
			CreateChange(changes, PropertyAdded, v3.RequiredLabel,
				nil, rSchema.Required.Value[k[g]].GetValueNode(), true, nil,
				rSchema.Required.Value[k[g]].GetValue)
		}
	}
	for g := range j {
		if _, ok := k[g]; !ok {
			CreateChange(changes, PropertyRemoved, v3.RequiredLabel,
				lSchema.Required.Value[j[g]].GetValueNode(), nil, true, lSchema.Required.Value[j[g]].GetValue,
				nil)
		}
	}

	// Enums
	j = make(map[string]int)
	k = make(map[string]int)
	for i := range lSchema.Enum.Value {
		j[toString(lSchema.Enum.Value[i].Value)] = i
	}
	for i := range rSchema.Enum.Value {
		k[toString(rSchema.Enum.Value[i].Value)] = i
	}
	for g := range k {
		if _, ok := j[g]; !ok {
			CreateChange(changes, PropertyAdded, v3.EnumLabel,
				nil, rSchema.Enum.Value[k[g]].GetValueNode(), false, nil,
				rSchema.Enum.Value[k[g]].GetValue)
		}
	}
	for g := range j {
		if _, ok := k[g]; !ok {
			CreateChange(changes, PropertyRemoved, v3.EnumLabel,
				lSchema.Enum.Value[j[g]].GetValueNode(), nil, true, lSchema.Enum.Value[j[g]].GetValue,
				nil)
		}
	}

	// Discriminator
	if lSchema.Discriminator.Value != nil && rSchema.Discriminator.Value != nil {
		// check if hash matches, if not then compare.
		if lSchema.Discriminator.Value.Hash() != rSchema.Discriminator.Value.Hash() {
			sc.DiscriminatorChanges = CompareDiscriminator(lSchema.Discriminator.Value, rSchema.Discriminator.Value)
		}
	}
	// added Discriminator
	if lSchema.Discriminator.Value == nil && rSchema.Discriminator.Value != nil {
		CreateChange(changes, ObjectAdded, v3.DiscriminatorLabel,
			nil, rSchema.Discriminator.ValueNode, true, nil, rSchema.Discriminator.Value)
	}
	// removed Discriminator
	if lSchema.Discriminator.Value != nil && rSchema.Discriminator.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.DiscriminatorLabel,
			lSchema.Discriminator.ValueNode, nil, true, lSchema.Discriminator.Value, nil)
	}

	// ExternalDocs
	if lSchema.ExternalDocs.Value != nil && rSchema.ExternalDocs.Value != nil {
		// check if hash matches, if not then compare.
		if lSchema.ExternalDocs.Value.Hash() != rSchema.ExternalDocs.Value.Hash() {
			sc.ExternalDocChanges = CompareExternalDocs(lSchema.ExternalDocs.Value, rSchema.ExternalDocs.Value)
		}
	}
	// added ExternalDocs
	if lSchema.ExternalDocs.Value == nil && rSchema.ExternalDocs.Value != nil {
		CreateChange(changes, ObjectAdded, v3.ExternalDocsLabel,
			nil, rSchema.ExternalDocs.ValueNode, false, nil, rSchema.ExternalDocs.Value)
	}
	// removed ExternalDocs
	if lSchema.ExternalDocs.Value != nil && rSchema.ExternalDocs.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.ExternalDocsLabel,
			lSchema.ExternalDocs.ValueNode, nil, false, lSchema.ExternalDocs.Value, nil)
	}

	// 3.1 properties
	// If
	if lSchema.If.Value != nil && rSchema.If.Value != nil {
		if !low.AreEqual(lSchema.If.Value, rSchema.If.Value) {
			sc.IfChanges = CompareSchemas(lSchema.If.Value, rSchema.If.Value)
		}
	}
	// added If
	if lSchema.If.Value == nil && rSchema.If.Value != nil {
		CreateChange(changes, ObjectAdded, v3.IfLabel,
			nil, rSchema.If.ValueNode, true, nil, rSchema.If.Value)
	}
	// removed If
	if lSchema.If.Value != nil && rSchema.If.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.IfLabel,
			lSchema.If.ValueNode, nil, true, lSchema.If.Value, nil)
	}
	// Else
	if lSchema.Else.Value != nil && rSchema.Else.Value != nil {
		if !low.AreEqual(lSchema.Else.Value, rSchema.Else.Value) {
			sc.ElseChanges = CompareSchemas(lSchema.Else.Value, rSchema.Else.Value)
		}
	}
	// added Else
	if lSchema.Else.Value == nil && rSchema.Else.Value != nil {
		CreateChange(changes, ObjectAdded, v3.ElseLabel,
			nil, rSchema.Else.ValueNode, true, nil, rSchema.Else.Value)
	}
	// removed Else
	if lSchema.Else.Value != nil && rSchema.Else.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.ElseLabel,
			lSchema.Else.ValueNode, nil, true, lSchema.Else.Value, nil)
	}
	// Then
	if lSchema.Then.Value != nil && rSchema.Then.Value != nil {
		if !low.AreEqual(lSchema.Then.Value, rSchema.Then.Value) {
			sc.ThenChanges = CompareSchemas(lSchema.Then.Value, rSchema.Then.Value)
		}
	}
	// added Then
	if lSchema.Then.Value == nil && rSchema.Then.Value != nil {
		CreateChange(changes, ObjectAdded, v3.ThenLabel,
			nil, rSchema.Then.ValueNode, true, nil, rSchema.Then.Value)
	}
	// removed Then
	if lSchema.Then.Value != nil && rSchema.Then.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.ThenLabel,
			lSchema.Then.ValueNode, nil, true, lSchema.Then.Value, nil)
	}
	// PropertyNames
	if lSchema.PropertyNames.Value != nil && rSchema.PropertyNames.Value != nil {
		if !low.AreEqual(lSchema.PropertyNames.Value, rSchema.PropertyNames.Value) {
			sc.PropertyNamesChanges = CompareSchemas(lSchema.PropertyNames.Value, rSchema.PropertyNames.Value)
		}
	}
	// added PropertyNames
	if lSchema.PropertyNames.Value == nil && rSchema.PropertyNames.Value != nil {
		CreateChange(changes, ObjectAdded, v3.PropertyNamesLabel,
			nil, rSchema.PropertyNames.ValueNode, true, nil, rSchema.PropertyNames.Value)
	}
	// removed PropertyNames
	if lSchema.PropertyNames.Value != nil && rSchema.PropertyNames.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.PropertyNamesLabel,
			lSchema.PropertyNames.ValueNode, nil, true, lSchema.PropertyNames.Value, nil)
	}
	// Contains
	if lSchema.Contains.Value != nil && rSchema.Contains.Value != nil {
		if !low.AreEqual(lSchema.Contains.Value, rSchema.Contains.Value) {
			sc.ContainsChanges = CompareSchemas(lSchema.Contains.Value, rSchema.Contains.Value)
		}
	}
	// added Contains
	if lSchema.Contains.Value == nil && rSchema.Contains.Value != nil {
		CreateChange(changes, ObjectAdded, v3.ContainsLabel,
			nil, rSchema.Contains.ValueNode, true, nil, rSchema.Contains.Value)
	}
	// removed Contains
	if lSchema.Contains.Value != nil && rSchema.Contains.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.ContainsLabel,
			lSchema.Contains.ValueNode, nil, true, lSchema.Contains.Value, nil)
	}
	// UnevaluatedItems
	if lSchema.UnevaluatedItems.Value != nil && rSchema.UnevaluatedItems.Value != nil {
		if !low.AreEqual(lSchema.UnevaluatedItems.Value, rSchema.UnevaluatedItems.Value) {
			sc.UnevaluatedItemsChanges = CompareSchemas(lSchema.UnevaluatedItems.Value, rSchema.UnevaluatedItems.Value)
		}
	}
	// added UnevaluatedItems
	if lSchema.UnevaluatedItems.Value == nil && rSchema.UnevaluatedItems.Value != nil {
		CreateChange(changes, ObjectAdded, v3.UnevaluatedItemsLabel,
			nil, rSchema.UnevaluatedItems.ValueNode, true, nil, rSchema.UnevaluatedItems.Value)
	}
	// removed UnevaluatedItems
	if lSchema.UnevaluatedItems.Value != nil && rSchema.UnevaluatedItems.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.UnevaluatedItemsLabel,
			lSchema.UnevaluatedItems.ValueNode, nil, true, lSchema.UnevaluatedItems.Value, nil)
	}

	// UnevaluatedProperties
	if lSchema.UnevaluatedProperties.Value != nil && rSchema.UnevaluatedProperties.Value != nil {
		if lSchema.UnevaluatedProperties.Value.IsA() && rSchema.UnevaluatedProperties.Value.IsA() {
			if !low.AreEqual(lSchema.UnevaluatedProperties.Value.A, rSchema.UnevaluatedProperties.Value.A) {
				sc.UnevaluatedPropertiesChanges = CompareSchemas(lSchema.UnevaluatedProperties.Value.A, rSchema.UnevaluatedProperties.Value.A)
			}
		} else {
			if lSchema.UnevaluatedProperties.Value.IsB() && rSchema.UnevaluatedProperties.Value.IsB() {
				if lSchema.UnevaluatedProperties.Value.B != rSchema.UnevaluatedProperties.Value.B {
					CreateChange(changes, Modified, v3.UnevaluatedPropertiesLabel,
						lSchema.UnevaluatedProperties.ValueNode, rSchema.UnevaluatedProperties.ValueNode, true,
						lSchema.UnevaluatedProperties.Value.B, rSchema.UnevaluatedProperties.Value.B)
				}
			} else {
				CreateChange(changes, Modified, v3.UnevaluatedPropertiesLabel,
					lSchema.UnevaluatedProperties.ValueNode, rSchema.UnevaluatedProperties.ValueNode, true,
					lSchema.UnevaluatedProperties.Value.B, rSchema.UnevaluatedProperties.Value.B)
			}
		}
	}

	// added UnevaluatedProperties
	if lSchema.UnevaluatedProperties.Value == nil && rSchema.UnevaluatedProperties.Value != nil {
		CreateChange(changes, ObjectAdded, v3.UnevaluatedPropertiesLabel,
			nil, rSchema.UnevaluatedProperties.ValueNode, true, nil, rSchema.UnevaluatedProperties.Value)
	}
	// removed UnevaluatedProperties
	if lSchema.UnevaluatedProperties.Value != nil && rSchema.UnevaluatedProperties.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.UnevaluatedPropertiesLabel,
			lSchema.UnevaluatedProperties.ValueNode, nil, true, lSchema.UnevaluatedProperties.Value, nil)
	}

	// Not
	if lSchema.Not.Value != nil && rSchema.Not.Value != nil {
		if !low.AreEqual(lSchema.Not.Value, rSchema.Not.Value) {
			sc.NotChanges = CompareSchemas(lSchema.Not.Value, rSchema.Not.Value)
		}
	}
	// added Not
	if lSchema.Not.Value == nil && rSchema.Not.Value != nil {
		CreateChange(changes, ObjectAdded, v3.NotLabel,
			nil, rSchema.Not.ValueNode, true, nil, rSchema.Not.Value)
	}
	// removed not
	if lSchema.Not.Value != nil && rSchema.Not.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.NotLabel,
			lSchema.Not.ValueNode, nil, true, lSchema.Not.Value, nil)
	}

	// items
	if lSchema.Items.Value != nil && rSchema.Items.Value != nil {
		if lSchema.Items.Value.IsA() && rSchema.Items.Value.IsA() {
			if !low.AreEqual(lSchema.Items.Value.A, rSchema.Items.Value.A) {
				sc.ItemsChanges = CompareSchemas(lSchema.Items.Value.A, rSchema.Items.Value.A)
			}
		} else {
			CreateChange(changes, Modified, v3.ItemsLabel,
				lSchema.Items.ValueNode, rSchema.Items.ValueNode, true, lSchema.Items.Value.B, rSchema.Items.Value.B)
		}
	}
	// added Items
	if lSchema.Items.Value == nil && rSchema.Items.Value != nil {
		CreateChange(changes, ObjectAdded, v3.ItemsLabel,
			nil, rSchema.Items.ValueNode, true, nil, rSchema.Items.Value)
	}
	// removed Items
	if lSchema.Items.Value != nil && rSchema.Items.Value == nil {
		CreateChange(changes, ObjectRemoved, v3.ItemsLabel,
			lSchema.Items.ValueNode, nil, true, lSchema.Items.Value, nil)
	}

	// check extensions
	sc.ExtensionChanges = CompareExtensions(lSchema.Extensions, rSchema.Extensions)

	// check core properties
	CheckProperties(props)
}

func checkExamples(lSchema *base.Schema, rSchema *base.Schema, changes *[]*Change) {
	// check examples (3.1+)
	var lExampKey, rExampKey []string
	lExampN := make(map[string]*yaml.Node)
	rExampN := make(map[string]*yaml.Node)
	lExampVal := make(map[string]any)
	rExampVal := make(map[string]any)

	// create keys by hashing values
	if lSchema.Examples.ValueNode != nil {
		for i := range lSchema.Examples.ValueNode.Content {
			key := low.GenerateHashString(lSchema.Examples.ValueNode.Content[i].Value)
			lExampKey = append(lExampKey, key)
			lExampVal[key] = lSchema.Examples.ValueNode.Content[i].Value
			lExampN[key] = lSchema.Examples.ValueNode.Content[i]

		}
	}
	if rSchema.Examples.ValueNode != nil {
		for i := range rSchema.Examples.ValueNode.Content {
			key := low.GenerateHashString(rSchema.Examples.ValueNode.Content[i].Value)
			rExampKey = append(rExampKey, key)
			rExampVal[key] = rSchema.Examples.ValueNode.Content[i].Value
			rExampN[key] = rSchema.Examples.ValueNode.Content[i]
		}
	}

	// if examples equal lengths, check for equality
	if len(lExampKey) == len(rExampKey) {
		for i := range lExampKey {
			if lExampKey[i] != rExampKey[i] {
				CreateChange(changes, Modified, v3.ExamplesLabel,
					lExampN[lExampKey[i]], rExampN[rExampKey[i]], false,
					lExampVal[lExampKey[i]], rExampVal[rExampKey[i]])
			}
		}
	}
	// examples were removed.
	if len(lExampKey) > len(rExampKey) {
		for i := range lExampKey {
			if i < len(rExampKey) && lExampKey[i] != rExampKey[i] {
				CreateChange(changes, Modified, v3.ExamplesLabel,
					lExampN[lExampKey[i]], rExampN[rExampKey[i]], false,
					lExampVal[lExampKey[i]], rExampVal[rExampKey[i]])
			}
			if i >= len(rExampKey) {
				CreateChange(changes, ObjectRemoved, v3.ExamplesLabel,
					lExampN[lExampKey[i]], nil, false,
					lExampVal[lExampKey[i]], nil)
			}
		}
	}

	// examples were added
	if len(lExampKey) < len(rExampKey) {
		for i := range rExampKey {
			if i < len(lExampKey) && lExampKey[i] != rExampKey[i] {
				CreateChange(changes, Modified, v3.ExamplesLabel,
					lExampN[lExampKey[i]], rExampN[rExampKey[i]], false,
					lExampVal[lExampKey[i]], rExampVal[rExampKey[i]])
			}
			if i >= len(lExampKey) {
				CreateChange(changes, ObjectAdded, v3.ExamplesLabel,
					nil, rExampN[rExampKey[i]], false,
					nil, rExampVal[rExampKey[i]])
			}
		}
	}
}

func extractSchemaChanges(
	lSchema []low.ValueReference[*base.SchemaProxy],
	rSchema []low.ValueReference[*base.SchemaProxy],
	label string,
	sc *[]*SchemaChanges,
	changes *[]*Change,
	done chan bool,
) {
	// if there is nothing here, there is nothing to do.
	if lSchema == nil && rSchema == nil {
		done <- true
		return
	}

	x := "%x"
	// create hash key maps to check equality
	lKeys := make([]string, 0, len(lSchema))
	rKeys := make([]string, 0, len(rSchema))
	lEntities := make(map[string]*base.SchemaProxy)
	rEntities := make(map[string]*base.SchemaProxy)
	for h := range lSchema {
		q := lSchema[h].Value
		z := fmt.Sprintf(x, q.Hash())
		lKeys = append(lKeys, z)
		lEntities[z] = q

	}
	for h := range rSchema {
		q := rSchema[h].Value
		z := fmt.Sprintf(x, q.Hash())
		rKeys = append(rKeys, z)
		rEntities[z] = q

	}

	// sort slices so that like for like is all sequenced.
	sort.Strings(lKeys)
	sort.Strings(rKeys)

	// check for identical lengths
	if len(lKeys) == len(rKeys) {
		for w := range lKeys {
			// keys are different, which means there are changes.
			if lKeys[w] != rKeys[w] {
				*sc = append(*sc, CompareSchemas(lEntities[lKeys[w]], rEntities[rKeys[w]]))
			}
		}
	}

	// things were removed
	if len(lKeys) > len(rKeys) {
		for w := range lKeys {
			if w < len(rKeys) && lKeys[w] != rKeys[w] {
				*sc = append(*sc, CompareSchemas(lEntities[lKeys[w]], rEntities[rKeys[w]]))
			}
			if w >= len(rKeys) {
				CreateChange(changes, ObjectRemoved, label,
					lEntities[lKeys[w]].GetValueNode(), nil, true, lEntities[lKeys[w]], nil)
			}
		}
	}

	// things were added
	if len(rKeys) > len(lKeys) {
		for w := range rKeys {
			if w < len(lKeys) && rKeys[w] != lKeys[w] {
				*sc = append(*sc, CompareSchemas(lEntities[lKeys[w]], rEntities[rKeys[w]]))
			}
			if w >= len(lKeys) {
				CreateChange(changes, ObjectAdded, label,
					nil, rEntities[rKeys[w]].GetValueNode(), false, nil, rEntities[rKeys[w]])
			}
		}
	}
	done <- true
}

// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"sort"
	"sync"
)

type SchemaChanges struct {
	PropertyChanges[*base.Schema]
	DiscriminatorChanges  *DiscriminatorChanges
	AllOfChanges          []*SchemaChanges
	AnyOfChanges          []*SchemaChanges
	OneOfChanges          []*SchemaChanges
	NotChanges            []*SchemaChanges
	ItemsChanges          []*SchemaChanges
	SchemaPropertyChanges map[string]*SchemaChanges
	ExternalDocChanges    *ExternalDocChanges
	ExtensionChanges      *ExtensionChanges
}

func (s *SchemaChanges) TotalChanges() int {
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
			t += s.AnyOfChanges[n].TotalChanges()
		}
	}
	if len(s.OneOfChanges) > 0 {
		for n := range s.OneOfChanges {
			t += s.OneOfChanges[n].TotalChanges()
		}
	}
	if len(s.NotChanges) > 0 {
		for n := range s.NotChanges {
			t += s.NotChanges[n].TotalChanges()
		}
	}
	if len(s.ItemsChanges) > 0 {
		for n := range s.ItemsChanges {
			t += s.ItemsChanges[n].TotalChanges()
		}
	}
	if s.SchemaPropertyChanges != nil {
		for n := range s.SchemaPropertyChanges {
			t += s.SchemaPropertyChanges[n].TotalChanges()
		}
	}
	if s.ExternalDocChanges != nil {
		t += s.ExternalDocChanges.TotalChanges()
	}
	if s.ExtensionChanges != nil {
		t += s.ExtensionChanges.TotalChanges()
	}
	return t
}

func (s *SchemaChanges) TotalBreakingChanges() int {
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
	if len(s.NotChanges) > 0 {
		for n := range s.NotChanges {
			t += s.NotChanges[n].TotalBreakingChanges()
		}
	}
	if len(s.ItemsChanges) > 0 {
		for n := range s.ItemsChanges {
			t += s.ItemsChanges[n].TotalBreakingChanges()
		}
	}
	if s.SchemaPropertyChanges != nil {
		for n := range s.SchemaPropertyChanges {
			t += s.SchemaPropertyChanges[n].TotalBreakingChanges()
		}
	}
	if s.ExternalDocChanges != nil {
		t += s.ExternalDocChanges.TotalBreakingChanges()
	}
	if s.ExtensionChanges != nil {
		t += s.ExtensionChanges.TotalBreakingChanges()
	}
	return t
}

func CompareSchemas(l, r *base.SchemaProxy) *SchemaChanges {
	sc := new(SchemaChanges)
	var changes []*Change[*base.Schema]

	// Added
	if l == nil && r != nil {
		CreateChange[*base.Schema](&changes, ObjectAdded, v3.SchemaLabel,
			nil, nil, true, nil, r)
		sc.Changes = changes
	}

	// Removed
	if l != nil && r == nil {
		CreateChange[*base.Schema](&changes, ObjectRemoved, v3.SchemaLabel,
			nil, nil, true, l, nil)
		sc.Changes = changes
	}

	if l != nil && r != nil {

		// if left proxy is a reference and right is a reference (we won't recurse into them)
		if l.IsSchemaReference() && r.IsSchemaReference() {
			// points to the same schema
			if l.GetSchemaReference() == r.GetSchemaReference() {
				// there is nothing to be done at this point.
				return nil
			} else {
				// references are different, that's all we care to know.
				CreateChange[*base.Schema](&changes, Modified, v3.RefLabel,
					l.GetValueNode().Content[1], r.GetValueNode().Content[1], true, l.GetSchemaReference(),
					r.GetSchemaReference())
				sc.Changes = changes
				return sc
			}
		}

		// changed from inline to ref
		if !l.IsSchemaReference() && r.IsSchemaReference() {
			CreateChange[*base.Schema](&changes, Modified, v3.RefLabel,
				l.GetValueNode(), r.GetValueNode().Content[1], true, l, r.GetSchemaReference())
			sc.Changes = changes
			return sc // we're done here
		}

		// changed from ref to inline
		if l.IsSchemaReference() && !r.IsSchemaReference() {
			CreateChange[*base.Schema](&changes, Modified, v3.RefLabel,
				l.GetValueNode().Content[1], r.GetValueNode(), true, l.GetSchemaReference(), r)
			sc.Changes = changes
			return sc // done, nothing else to do.
		}

		lSchema := l.Schema()
		rSchema := r.Schema()

		if low.AreEqual(lSchema, rSchema) {
			// there is no point going on, we know nothing changed!
			return nil
		}

		var props []*PropertyCheck[*base.Schema]

		// $schema (breaking change)
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.SchemaTypeRef.ValueNode,
			RightNode: rSchema.SchemaTypeRef.ValueNode,
			Label:     v3.SchemaDialectLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// ExclusiveMaximum
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.ExclusiveMaximum.ValueNode,
			RightNode: rSchema.ExclusiveMaximum.ValueNode,
			Label:     v3.ExclusiveMaximumLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// ExclusiveMinimum
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.ExclusiveMinimum.ValueNode,
			RightNode: rSchema.ExclusiveMinimum.ValueNode,
			Label:     v3.ExclusiveMinimumLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// Type
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Type.ValueNode,
			RightNode: rSchema.Type.ValueNode,
			Label:     v3.TypeLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// Title
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Title.ValueNode,
			RightNode: rSchema.Title.ValueNode,
			Label:     v3.TitleLabel,
			Changes:   &changes,
			Breaking:  false,
			Original:  lSchema,
			New:       rSchema,
		})

		// MultipleOf
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.MultipleOf.ValueNode,
			RightNode: rSchema.MultipleOf.ValueNode,
			Label:     v3.MultipleOfLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// Maximum
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Maximum.ValueNode,
			RightNode: rSchema.Maximum.ValueNode,
			Label:     v3.MaximumLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// Minimum
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Minimum.ValueNode,
			RightNode: rSchema.Minimum.ValueNode,
			Label:     v3.MinimumLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// MaxLength
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.MaxLength.ValueNode,
			RightNode: rSchema.MaxLength.ValueNode,
			Label:     v3.MaxLengthLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// MinLength
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.MinLength.ValueNode,
			RightNode: rSchema.MinLength.ValueNode,
			Label:     v3.MinLengthLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// Pattern
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Pattern.ValueNode,
			RightNode: rSchema.Pattern.ValueNode,
			Label:     v3.PatternLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// Format
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Format.ValueNode,
			RightNode: rSchema.Format.ValueNode,
			Label:     v3.FormatLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// MaxItems
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.MaxItems.ValueNode,
			RightNode: rSchema.MaxItems.ValueNode,
			Label:     v3.MaxItemsLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// MinItems
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.MinItems.ValueNode,
			RightNode: rSchema.MinItems.ValueNode,
			Label:     v3.MinItemsLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// MaxProperties
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.MaxProperties.ValueNode,
			RightNode: rSchema.MaxProperties.ValueNode,
			Label:     v3.MaxPropertiesLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// MinProperties
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.MinProperties.ValueNode,
			RightNode: rSchema.MinProperties.ValueNode,
			Label:     v3.MinPropertiesLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// UniqueItems
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.UniqueItems.ValueNode,
			RightNode: rSchema.UniqueItems.ValueNode,
			Label:     v3.UniqueItemsLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// AdditionalProperties
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.AdditionalProperties.ValueNode,
			RightNode: rSchema.AdditionalProperties.ValueNode,
			Label:     v3.AdditionalPropertiesLabel,
			Changes:   &changes,
			Breaking:  false,
			Original:  lSchema,
			New:       rSchema,
		})

		// Description
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Description.ValueNode,
			RightNode: rSchema.Description.ValueNode,
			Label:     v3.DescriptionLabel,
			Changes:   &changes,
			Breaking:  false,
			Original:  lSchema,
			New:       rSchema,
		})

		// ContentEncoding
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.ContentEncoding.ValueNode,
			RightNode: rSchema.ContentEncoding.ValueNode,
			Label:     v3.ContentEncodingLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// ContentMediaType
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.ContentMediaType.ValueNode,
			RightNode: rSchema.ContentMediaType.ValueNode,
			Label:     v3.ContentMediaType,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// Default
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Default.ValueNode,
			RightNode: rSchema.Default.ValueNode,
			Label:     v3.DefaultLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// Nullable
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Nullable.ValueNode,
			RightNode: rSchema.Nullable.ValueNode,
			Label:     v3.NullableLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// ReadOnly
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.ReadOnly.ValueNode,
			RightNode: rSchema.ReadOnly.ValueNode,
			Label:     v3.ReadOnlyLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// WriteOnly
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.WriteOnly.ValueNode,
			RightNode: rSchema.WriteOnly.ValueNode,
			Label:     v3.WriteOnlyLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// Example
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Example.ValueNode,
			RightNode: rSchema.Example.ValueNode,
			Label:     v3.ExampleLabel,
			Changes:   &changes,
			Breaking:  false,
			Original:  lSchema,
			New:       rSchema,
		})

		// Deprecated
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Deprecated.ValueNode,
			RightNode: rSchema.Deprecated.ValueNode,
			Label:     v3.DeprecatedLabel,
			Changes:   &changes,
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
				CreateChange[*base.Schema](&changes, PropertyAdded, v3.RequiredLabel,
					nil, rSchema.Required.Value[k[g]].GetValueNode(), true, nil,
					rSchema.Required.Value[k[g]].GetValue)
			}
		}
		for g := range j {
			if _, ok := k[g]; !ok {
				CreateChange[*base.Schema](&changes, PropertyRemoved, v3.RequiredLabel,
					lSchema.Required.Value[j[g]].GetValueNode(), nil, true, lSchema.Required.Value[j[g]].GetValue,
					nil)
			}
		}

		// Enums
		j = make(map[string]int)
		k = make(map[string]int)
		for i := range lSchema.Enum.Value {
			j[lSchema.Enum.Value[i].Value] = i
		}
		for i := range rSchema.Enum.Value {
			k[rSchema.Enum.Value[i].Value] = i
		}
		for g := range k {
			if _, ok := j[g]; !ok {
				CreateChange[*base.Schema](&changes, PropertyAdded, v3.EnumLabel,
					nil, rSchema.Enum.Value[k[g]].GetValueNode(), false, nil,
					rSchema.Enum.Value[k[g]].GetValue)
			}
		}
		for g := range j {
			if _, ok := k[g]; !ok {
				CreateChange[*base.Schema](&changes, PropertyRemoved, v3.EnumLabel,
					lSchema.Enum.Value[j[g]].GetValueNode(), nil, true, lSchema.Enum.Value[j[g]].GetValue,
					nil)
			}
		}

		// check core properties
		CheckProperties(props)

		propChanges := make(map[string]*SchemaChanges)

		lProps := make([]string, len(lSchema.Properties.Value))
		lEntities := make(map[string]*base.SchemaProxy)
		rProps := make([]string, len(rSchema.Properties.Value))
		rEntities := make(map[string]*base.SchemaProxy)

		for w := range lSchema.Properties.Value {
			if !lSchema.Properties.Value[w].Value.IsSchemaReference() {
				lProps = append(lProps, w.Value)
				lEntities[w.Value] = lSchema.Properties.Value[w].Value
			}
		}
		for w := range rSchema.Properties.Value {
			if !rSchema.Properties.Value[w].Value.IsSchemaReference() {
				rProps = append(rProps, w.Value)
				rEntities[w.Value] = rSchema.Properties.Value[w].Value
			}
		}
		sort.Strings(lProps)
		sort.Strings(rProps)

		var propLock sync.Mutex
		checkProperty := func(key string, lp, rp *base.SchemaProxy, propChanges map[string]*SchemaChanges, done chan bool) {
			if lp != nil && rp != nil {
				ls := lp.Schema()
				rs := rp.Schema()
				if low.AreEqual(ls, rs) {
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

		doneChan := make(chan bool)
		totalProperties := 0
		if len(lProps) == len(rProps) {
			for w := range lProps {
				lp := lEntities[lProps[w]]
				rp := rEntities[rProps[w]]
				if lProps[w] == rProps[w] && lp != nil && rp != nil {
					totalProperties++
					go checkProperty(lProps[w], lp, rp, propChanges, doneChan)
				}

				// keys do not match, even after sorting, means a like for like replacement.
				if lProps[w] != rProps[w] {

					// old removed, new added.
					CreateChange[*base.Schema](&changes, ObjectAdded, v3.PropertiesLabel,
						nil, rEntities[rProps[w]].GetValueNode(), false, nil, rEntities[rProps[w]])
					CreateChange[*base.Schema](&changes, ObjectRemoved, v3.PropertiesLabel,
						lEntities[lProps[w]].GetValueNode(), nil, true, lEntities[lProps[w]], nil)
				}

			}
		}

		// something removed
		if len(lProps) > len(rProps) {
			for w := range lProps {
				if w < len(rProps) {
					totalProperties++
					go checkProperty(lProps[w], lEntities[lProps[w]], rEntities[rProps[w]], propChanges, doneChan)
				}
				if w >= len(rProps) {
					CreateChange[*base.Schema](&changes, ObjectRemoved, v3.PropertiesLabel,
						lEntities[lProps[w]].GetValueNode(), nil, true, lEntities[lProps[w]], nil)
				}
			}
		}

		// something added
		if len(rProps) > len(lProps) {
			for w := range rProps {
				if w < len(lProps) {
					totalProperties++
					go checkProperty(rProps[w], lEntities[lProps[w]], rEntities[rProps[w]], propChanges, doneChan)
				}
				if w >= len(rProps) {
					CreateChange[*base.Schema](&changes, ObjectAdded, v3.PropertiesLabel,
						nil, rEntities[rProps[w]].GetValueNode(), false, nil, rEntities[rProps[w]])
				}
			}
		}

		sc.SchemaPropertyChanges = propChanges

		// check polymorphic and multi-values async for speed.
		go extractSchemaChanges(lSchema.OneOf.Value, rSchema.OneOf.Value, v3.OneOfLabel,
			&sc.OneOfChanges, &changes, doneChan)

		go extractSchemaChanges(lSchema.AllOf.Value, rSchema.AllOf.Value, v3.AllOfLabel,
			&sc.AllOfChanges, &changes, doneChan)

		go extractSchemaChanges(lSchema.AnyOf.Value, rSchema.AnyOf.Value, v3.AnyOfLabel,
			&sc.AnyOfChanges, &changes, doneChan)

		go extractSchemaChanges(lSchema.Items.Value, rSchema.Items.Value, v3.ItemsLabel,
			&sc.ItemsChanges, &changes, doneChan)

		go extractSchemaChanges(lSchema.Not.Value, rSchema.Not.Value, v3.ItemsLabel,
			&sc.ItemsChanges, &changes, doneChan)

		totalChecks := totalProperties + 5
		completedChecks := 0
		for completedChecks < totalChecks {
			select {
			case <-doneChan:
				completedChecks++
			}
		}

	}

	// done
	sc.Changes = changes
	if sc.TotalChanges() <= 0 {
		return nil
	}
	return sc

}

func extractSchemaChanges(
	lSchema []low.ValueReference[*base.SchemaProxy],
	rSchema []low.ValueReference[*base.SchemaProxy],
	label string,
	sc *[]*SchemaChanges,
	changes *[]*Change[*base.Schema],
	done chan bool) {

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
		if !q.IsSchemaReference() {
			w := q.Schema()
			z := fmt.Sprintf(x, w.Hash())
			lKeys = append(lKeys, z)
			lEntities[z] = q
		}
	}
	for h := range rSchema {
		q := rSchema[h].Value
		if !q.IsSchemaReference() {
			w := q.Schema()
			z := fmt.Sprintf(x, w.Hash())
			rKeys = append(rKeys, z)
			rEntities[z] = q
		}
	}

	if len(lKeys) <= 0 && len(rKeys) <= 0 {
		done <- true
		return
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
				CreateChange[*base.Schema](changes, ObjectRemoved, label,
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
				CreateChange[*base.Schema](changes, ObjectAdded, label,
					nil, rEntities[rKeys[w]].GetValueNode(), false, nil, rEntities[rKeys[w]])
			}
		}
	}
	done <- true
}

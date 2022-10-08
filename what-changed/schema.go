// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

type SchemaChanges struct {
	PropertyChanges[*base.Schema]
	DiscriminatorChanges  *DiscriminatorChanges
	AllOfChanges          []*SchemaChanges
	AnyOfChanges          *SchemaChanges
	NotChanges            *SchemaChanges
	ItemsChanges          *SchemaChanges
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
	if s.AnyOfChanges != nil {
		t += s.AnyOfChanges.TotalChanges()
	}
	if s.NotChanges != nil {
		t += s.NotChanges.TotalChanges()
	}
	if s.ItemsChanges != nil {
		t += s.ItemsChanges.TotalChanges()
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
	if s.AnyOfChanges != nil {
		t += s.AnyOfChanges.TotalBreakingChanges()
	}
	if s.NotChanges != nil {
		t += s.NotChanges.TotalBreakingChanges()
	}
	if s.ItemsChanges != nil {
		t += s.ItemsChanges.TotalBreakingChanges()
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
					nil, nil, true, l.GetSchemaReference(), r.GetSchemaReference())
				sc.Changes = changes
				return sc
			}
		}

		// changed from ref to inline
		if !l.IsSchemaReference() && r.IsSchemaReference() {
			CreateChange[*base.Schema](&changes, Modified, v3.RefLabel,
				nil, nil, false, "", r.GetSchemaReference())
			sc.Changes = changes
			return sc // we're done here
		}

		// changed from inline to ref
		if l.IsSchemaReference() && !r.IsSchemaReference() {
			CreateChange[*base.Schema](&changes, Modified, v3.RefLabel,
				nil, nil, false, l.GetSchemaReference(), "")
			sc.Changes = changes
			return sc // done, nothing else to do.
		}

		lSchema := l.Schema()
		rSchema := r.Schema()

		leftHash := lSchema.Hash()
		rightHash := rSchema.Hash()

		fmt.Printf("%v-%v", leftHash, rightHash)

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

		// UniqueItems
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.UniqueItems.ValueNode,
			RightNode: rSchema.UniqueItems.ValueNode,
			Label:     v3.MinLengthLabel,
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

		// Required
		j := make(map[string]int)
		k := make(map[string]int)
		for i := range lSchema.Required.Value {
			j[lSchema.Required.Value[i].Value] = i
		}
		for i := range rSchema.Required.Value {
			k[rSchema.Required.Value[i].Value] = i
		}

		// added
		for g := range k {
			if _, ok := j[g]; !ok {
				CreateChange[*base.Schema](&changes, PropertyAdded, v3.RequiredLabel,
					nil, rSchema.Required.Value[k[g]].GetValueNode(), true, nil,
					rSchema.Required.Value[k[g]].GetValue)
			}
		}
		// removed
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

		// added
		for g := range k {
			if _, ok := j[g]; !ok {
				CreateChange[*base.Schema](&changes, PropertyAdded, v3.EnumLabel,
					nil, rSchema.Enum.Value[k[g]].GetValueNode(), false, nil,
					rSchema.Enum.Value[k[g]].GetValue)
			}
		}
		// removed
		for g := range j {
			if _, ok := k[g]; !ok {
				CreateChange[*base.Schema](&changes, PropertyRemoved, v3.EnumLabel,
					lSchema.Enum.Value[j[g]].GetValueNode(), nil, true, lSchema.Enum.Value[j[g]].GetValue,
					nil)
			}
		}

		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Required.ValueNode,
			RightNode: rSchema.Required.ValueNode,
			Label:     v3.RequiredLabel,
			Changes:   &changes,
			Breaking:  true,
			Original:  lSchema,
			New:       rSchema,
		})

		// Enum
		props = append(props, &PropertyCheck[*base.Schema]{
			LeftNode:  lSchema.Enum.ValueNode,
			RightNode: rSchema.Enum.ValueNode,
			Label:     v3.EnumLabel,
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
		// TODO: end of re-do

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
			Label:     v3.MinLengthLabel,
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

		// check properties
		CheckProperties(props)

		// check objects.
		// AllOf
		// if both sides are equal
		//if len(rSchema.AllOf.Value) == len(lSchema.AllOf.Value) {
		var multiChange []*SchemaChanges
		for d := range lSchema.AllOf.Value {
			var lSch, rSch *base.SchemaProxy
			lSch = lSchema.AllOf.Value[d].Value

			if rSchema.AllOf.Value[d].Value != nil {
				rSch = rSchema.AllOf.Value[d].Value
			}
			// if neither is a reference, build the schema and compare.
			//if !lSch.IsSchemaReference() && !rSch.IsSchemaReference() {
			multiChange = append(multiChange, CompareSchemas(lSch, rSch))
			//}
			// if the left is a reference and right is inline, log a modification, but no recursion.
			//if lSch.IsSchemaReference() && !rSch.IsSchemaReference() {
			//	CreateChange[*base.Schema](&changes, Modified, v3.AllOfLabel,
			//		nil, nil, false, lSch.GetSchemaReference(), "")
			//}
			//
			//// if the right is a reference and left is inline, log a modification, but no recursion.
			//if !lSch.IsSchemaReference() && rSch.IsSchemaReference() {
			//	CreateChange[*base.Schema](&changes, Modified, v3.AllOfLabel,
			//		nil, nil, false, "", rSch.GetSchemaReference())
			//}

		}

		//}

		//check if the right is longer that the left (added)
		if len(rSchema.AllOf.Value) > len(lSchema.AllOf.Value) {
			y := len(lSchema.AllOf.Value)
			if y < 0 {
				y = 0
			}
			for s := range rSchema.AllOf.Value[y:] {
				rSch := rSchema.AllOf.Value[s].Value
				multiChange = append(multiChange, CompareSchemas(nil, rSch))

				//if !rSchema.AllOf.Value[s].Value.IsSchemaReference() {
				//	CreateChange[*base.Schema](&changes, ObjectAdded, v3.AllOfLabel,
				//		nil, rSchema.AllOf.Value[s].GetValueNode(), false, nil, rSchema.AllOf.Value[s].Value.Schema())
				//} else {
				//	CreateChange[*base.Schema](&changes, ObjectAdded, v3.AllOfLabel,
				//		nil, rSchema.AllOf.Value[s].GetValueNode(), false, nil, rSchema.AllOf.Value[s].Value)
				//}
			}

		}

		if len(multiChange) > 0 {
			sc.AllOfChanges = multiChange
		}

		//
		//// check if the left is longer that the right (removed)
		//if len(lSchema.AllOf.Value) > len(rSchema.AllOf.Value) {
		//	var multiChange []*SchemaChanges
		//	for s := range lSchema.AllOf.Value[len(rSchema.AllOf.Value)-1:] {
		//		lSch := lSchema.AllOf.Value[s].Value
		//		multiChange = append(multiChange, CompareSchemas(lSch, nil))
		//		//if !lSchema.AllOf.Value[s].Value.IsSchemaReference() {
		//		//	CreateChange[*base.Schema](&changes, ObjectRemoved, v3.AllOfLabel,
		//		//		lSchema.AllOf.Value[s].GetValueNode(), nil, false, lSchema.AllOf.Value[s].Value.Schema(), nil)
		//		//} else {
		//		//	CreateChange[*base.Schema](&changes, ObjectRemoved, v3.AllOfLabel,
		//		//		lSchema.AllOf.Value[s].GetValueNode(), nil, false, lSchema.AllOf.Value[s].Value, nil)
		//		//}
		//	}
		//	if len(multiChange) > 0 {
		//		sc.AllOfChanges = multiChange
		//	}
		//}

	}

	// done
	sc.Changes = changes
	if sc.TotalChanges() <= 0 {
		return nil
	}
	return sc

}

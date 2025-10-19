// Copyright 2022 Hugo Stijns
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/index"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func Test_CheckForModification(t *testing.T) {
	tests := []struct {
		name   string
		left   string
		right  string
		differ bool
	}{
		{"Same string quoted", `value`, `"value"`, false},
		{"Same string", `value`, `value`, false},
		{"Same boolean", `true`, `true`, false},
		{"Different boolean", `true`, `false`, true},
		{"Different string", `value_a`, `value_b`, true},
		{"Different int", `123`, `"123"`, true},
		{"Different float", `123.456`, `"123.456"`, true},
		{"Different boolean quoted", `true`, `"true"`, true},
		{"Different date", `2022-12-29`, `"2022-12-29"`, true},
		{"Different value and tag", `2.0`, `2.0.0`, true},
		{"From null to empty value", `null`, `""`, false},
		{"From empty value to null", `""`, `null`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lNode, rNode yaml.Node
			_ = yaml.Unmarshal([]byte(tt.left), &lNode)
			_ = yaml.Unmarshal([]byte(tt.right), &rNode)

			changes := []*Change{}
			CheckForModification(lNode.Content[0], rNode.Content[0], "test", &changes, false, "old", "new")

			if tt.differ {
				assert.Len(t, changes, 1)
			} else {
				assert.Empty(t, changes)
			}
		})
	}
}

func Test_CheckForModification_ArrayMap(t *testing.T) {
	tests := []struct {
		name   string
		left   any
		right  any
		differ bool
	}{
		{"Same slice", []string{"cake"}, []string{"cake"}, false},
		{"bigger slice right", []string{"cake"}, []string{"cake", "cheese"}, true},
		{"bigger slice left", []string{"cake", "burgers"}, []string{"cake"}, true},
		{"different slice left", "cake", []string{"cake"}, true},
		{"different slice right", []string{"cake"}, "cake", true},
		{"different slice value", []string{"cake"}, []string{"burgers"}, true},
		{"same map", map[string]string{"pie": "cake"}, map[string]string{"pie": "cake"}, false},
		{"different map", map[string]string{"pie": "cake"}, map[string]string{"pizza": "burgers"}, true},
		{"different map left", "pie", map[string]string{"pie": "cake"}, true},
		{"different map right", map[string]string{"pie": "cake"}, "burgers", true},
		{"different map", map[string]string{"pie": "cake"}, map[string]string{"pizza": "time"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lNode, rNode yaml.Node
			encA, _ := yaml.Marshal(tt.left)
			encB, _ := yaml.Marshal(tt.right)
			_ = yaml.Unmarshal(encA, &lNode)
			_ = yaml.Unmarshal(encB, &rNode)

			changes := []*Change{}
			CheckForModification(lNode.Content[0], rNode.Content[0], tt.name, &changes, false, "old", "new")

			if tt.differ {
				assert.Len(t, changes, 1)
			} else {
				assert.Empty(t, changes)
			}
		})
	}
}

func Test_CheckMapsRemoval(t *testing.T) {
	mapA := make(map[string]string)
	mapB := make(map[string]string)

	mapA["key"] = "value"
	mapB["key"] = "shmalue"

	tests := []struct {
		name   string
		left   map[string]string
		right  map[string]string
		differ bool
	}{
		{"Same map", mapA, mapB, false},
		{"Different map", mapA, mapB, false},
		{"Add map", nil, mapA, false},
		{"Remove map", mapA, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lNode, rNode yaml.Node
			encA, _ := yaml.Marshal(tt.left)
			encB, _ := yaml.Marshal(tt.right)
			_ = yaml.Unmarshal(encA, &lNode)
			_ = yaml.Unmarshal(encB, &rNode)

			changes := []*Change{}

			r := rNode.Content[0]
			if len(r.Content) == 0 {
				r = nil
			}
			CheckForRemoval(lNode.Content[0], r, "test", &changes, false, "old", "new")

			if tt.differ {
				assert.Len(t, changes, 1)
			} else {
				assert.Empty(t, changes)
			}
		})
	}
}

func Test_CheckMapsAddition(t *testing.T) {
	mapA := make(map[string]string)
	mapB := make(map[string]string)

	mapA["key"] = "value"
	mapB["key"] = "shmalue"

	tests := []struct {
		name   string
		left   map[string]string
		right  map[string]string
		differ bool
		empty  bool
	}{
		{"Same map", mapA, mapB, false, false},
		{"Different map", mapA, mapB, false, false},
		{"Add map", nil, mapA, true, false},
		{"Add map value", nil, mapB, true, true},
		{"Remove map", mapA, nil, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lNode, rNode yaml.Node
			encA, _ := yaml.Marshal(tt.left)
			encB, _ := yaml.Marshal(tt.right)
			_ = yaml.Unmarshal(encA, &lNode)
			_ = yaml.Unmarshal(encB, &rNode)

			changes := []*Change{}

			r := rNode.Content[0]
			l := lNode.Content[0]
			if len(r.Content) == 0 {
				r = nil
			}
			if len(l.Content) == 0 {
				if !tt.empty {
					l = nil
				}
			}
			CheckForAddition(l, r, "test", &changes, false, "old", "new")

			if tt.differ {
				assert.Len(t, changes, 1)
			} else {
				assert.Empty(t, changes)
			}
		})
	}
}

func TestFlattenLowLevelOrderedMap(t *testing.T) {
	m := orderedmap.New[low.KeyReference[string], low.ValueReference[string]]()
	m.Set(low.KeyReference[string]{
		Value: "key",
	}, low.ValueReference[string]{
		Value: "value",
	})

	o := FlattenLowLevelOrderedMap[string](m)
	assert.Equal(t, map[string]*low.ValueReference[string]{
		"key": {
			Value: "value",
		},
	}, o)
}

func TestCheckMapForAdditionRemoval(t *testing.T) {
	var changes []*Change

	l := orderedmap.New[low.KeyReference[string], low.ValueReference[string]]()
	l.Set(low.KeyReference[string]{
		Value: "key",
	}, low.ValueReference[string]{
		Value:     "value",
		ValueNode: utils.CreateStringNode("value"),
	})
	l.Set(low.KeyReference[string]{
		Value: "key2",
	}, low.ValueReference[string]{
		Value:     "value2",
		ValueNode: utils.CreateStringNode("value2"),
	})

	r := orderedmap.New[low.KeyReference[string], low.ValueReference[string]]()
	r.Set(low.KeyReference[string]{
		Value: "key",
	}, low.ValueReference[string]{
		Value:     "value",
		ValueNode: utils.CreateStringNode("value"),
	})

	CheckMapForAdditionRemoval(l, r, &changes, "label")
	assert.Len(t, changes, 1)
}

type test_hasIndex struct {
	index *index.SpecIndex
}

func (t *test_hasIndex) GetIndex() *index.SpecIndex {
	return t.index
}

func Test_checkLocation(t *testing.T) {

	idx := index.NewSpecIndex(&yaml.Node{Content: []*yaml.Node{{Content: []*yaml.Node{}}}})
	idxB := index.NewSpecIndex(&yaml.Node{Content: []*yaml.Node{{Content: []*yaml.Node{}}}})
	rolodex := index.NewRolodex(&index.SpecIndexConfig{})
	idx.SetRolodex(rolodex)
	idxB.SetRolodex(rolodex)
	rolodex.SetRootIndex(idx)

	testHasIndex := &test_hasIndex{
		index: idxB,
	}

	// https://suno.com/s/FtPAc2SaXEw5vTsH
	idxB.SetAbsolutePath("milly-milk-bottle")
	assert.True(t, checkLocation(&ChangeContext{DocumentLocation: "sunny-spain"}, testHasIndex))

}

func Test_checkLocation_sameIdx(t *testing.T) {

	idx := index.NewSpecIndex(&yaml.Node{Content: []*yaml.Node{{Content: []*yaml.Node{}}}})
	rolodex := index.NewRolodex(&index.SpecIndexConfig{})
	idx.SetRolodex(rolodex)
	rolodex.SetRootIndex(idx)

	testHasIndex := &test_hasIndex{
		index: idx,
	}
	idx.SetAbsolutePath("milly-milk-bottle")
	assert.False(t, checkLocation(&ChangeContext{DocumentLocation: ""}, testHasIndex))

}

// TestSetReferenceIfExists tests the SetReferenceIfExists function
func TestSetReferenceIfExists(t *testing.T) {
	tests := []struct {
		name           string
		setupValue     func() *low.ValueReference[string]
		setupChangeObj func() any
		expectRef      string
	}{
		{
			name: "sets reference when value has reference and object implements interface",
			setupValue: func() *low.ValueReference[string] {
				val := low.ValueReference[string]{
					Value: "test",
				}
				val.SetReference("#/components/headers/TestHeader", nil)
				return &val
			},
			setupChangeObj: func() any {
				return &PropertyChanges{}
			},
			expectRef: "#/components/headers/TestHeader",
		},
		{
			name: "does not set reference when value has no reference",
			setupValue: func() *low.ValueReference[string] {
				val := low.ValueReference[string]{
					Value: "test",
				}
				return &val
			},
			setupChangeObj: func() any {
				return &PropertyChanges{}
			},
			expectRef: "",
		},
		{
			name:       "handles nil value without panic",
			setupValue: func() *low.ValueReference[string] { return nil },
			setupChangeObj: func() any {
				return &PropertyChanges{}
			},
			expectRef: "",
		},
		{
			name: "handles non-ChangeIsReferenced object without panic",
			setupValue: func() *low.ValueReference[string] {
				val := low.ValueReference[string]{
					Value: "test",
				}
				val.SetReference("#/components/headers/TestHeader", nil)
				return &val
			},
			setupChangeObj: func() any {
				// Return something that doesn't implement ChangeIsReferenced
				return &struct{}{}
			},
			expectRef: "",
		},
		{
			name: "sets reference for remote reference",
			setupValue: func() *low.ValueReference[string] {
				val := low.ValueReference[string]{
					Value: "test",
				}
				val.SetReference("./schemas.yaml#/components/schemas/TestSchema", nil)
				return &val
			},
			setupChangeObj: func() any {
				return &PropertyChanges{}
			},
			expectRef: "./schemas.yaml#/components/schemas/TestSchema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := tt.setupValue()
			changeObj := tt.setupChangeObj()

			// Call the function
			SetReferenceIfExists(value, changeObj)

			// Check the result if it implements ChangeIsReferenced
			if refObj, ok := changeObj.(ChangeIsReferenced); ok {
				assert.Equal(t, tt.expectRef, refObj.GetChangeReference())
			}
		})
	}
}

// TestSetReferenceIfExists_Integration tests the integration with actual comparison scenarios
func TestSetReferenceIfExists_Integration(t *testing.T) {
	// Create a referenced value
	valueNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "test-value",
	}

	refValue := low.ValueReference[string]{
		Value:     "test-value",
		ValueNode: valueNode,
	}
	refValue.SetReference("#/components/parameters/TestParam", nil)

	// Test with HeaderChanges (which embeds PropertyChanges)
	t.Run("HeaderChanges with reference", func(t *testing.T) {
		headerChanges := &HeaderChanges{
			PropertyChanges: &PropertyChanges{},
		}

		SetReferenceIfExists(&refValue, headerChanges)
		assert.Equal(t, "#/components/parameters/TestParam", headerChanges.GetChangeReference())
	})

	// Test with SchemaChanges
	t.Run("SchemaChanges with reference", func(t *testing.T) {
		schemaChanges := &SchemaChanges{
			PropertyChanges: &PropertyChanges{},
		}

		SetReferenceIfExists(&refValue, schemaChanges)
		assert.Equal(t, "#/components/parameters/TestParam", schemaChanges.GetChangeReference())
	})

	// Test that it preserves existing references
	t.Run("preserves existing reference", func(t *testing.T) {
		changes := &PropertyChanges{
			ChangeReference: "#/existing/reference",
		}

		// Create a value without reference
		nonRefValue := low.ValueReference[string]{
			Value: "test",
		}

		SetReferenceIfExists(&nonRefValue, changes)
		// Should still have the existing reference
		assert.Equal(t, "#/existing/reference", changes.GetChangeReference())
	})

	// Test overwriting reference
	t.Run("overwrites reference when new reference exists", func(t *testing.T) {
		changes := &PropertyChanges{
			ChangeReference: "#/old/reference",
		}

		SetReferenceIfExists(&refValue, changes)
		// Should have the new reference
		assert.Equal(t, "#/components/parameters/TestParam", changes.GetChangeReference())
	})
}

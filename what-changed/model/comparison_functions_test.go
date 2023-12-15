// Copyright 2022 Hugo Stijns
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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

// Copyright 2022 Hugo Stijns
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

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

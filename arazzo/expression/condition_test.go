// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package expression

import "testing"

func TestSplitSimpleCondition(t *testing.T) {
	tests := []struct {
		input           string
		left, op, right string
		found           bool
	}{
		{input: "$statusCode == 201", left: "$statusCode", op: "==", right: "201", found: true},
		{input: "$response.body#/limits/>=minimum != true", left: "$response.body#/limits/>=minimum", op: "!=", right: "true", found: true},
		{input: "count >= 2", left: "count", op: ">=", right: "2", found: true},
		{input: "count <= 2", left: "count", op: "<=", right: "2", found: true},
		{input: "count > 2", left: "count", op: ">", right: "2", found: true},
		{input: "count < 2", left: "count", op: "<", right: "2", found: true},
		{input: "$statusCode", found: false},
		{input: "== 201", found: false},
		{input: "$statusCode ==", found: false},
		{input: "plain operand", found: false},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			left, op, right, found := SplitSimpleCondition(test.input)
			if left != test.left || op != test.op || right != test.right || found != test.found {
				t.Fatalf("SplitSimpleCondition(%q) = (%q, %q, %q, %t)", test.input, left, op, right, found)
			}
		})
	}
}

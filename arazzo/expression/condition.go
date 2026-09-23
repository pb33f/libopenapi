// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package expression

import "strings"

var simpleConditionOperators = []string{"==", "!=", ">=", "<=", ">", "<"}

// SplitSimpleCondition separates the operands and comparison operator in an
// Arazzo simple condition. Operators inside a runtime expression's JSON
// Pointer are ignored.
func SplitSimpleCondition(input string) (left, operator, right string, found bool) {
	searchStart := 0
	if strings.HasPrefix(input, "$") {
		if space := strings.IndexByte(input, ' '); space >= 0 {
			searchStart = space
		} else {
			return "", "", "", false
		}
	}
	for _, candidate := range simpleConditionOperators {
		if index := strings.Index(input[searchStart:], candidate); index >= 0 {
			index += searchStart
			left = strings.TrimSpace(input[:index])
			right = strings.TrimSpace(input[index+len(candidate):])
			if left == "" || right == "" {
				return "", "", "", false
			}
			return left, candidate, right, true
		}
	}
	return "", "", "", false
}

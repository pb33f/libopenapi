// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"fmt"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestCompareDocumentsRequiredDirection(t *testing.T) {
	const request = `"requestBody":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Widget"}}}},"responses":{"204":{"description":"ok"}}`
	const response = `"responses":{"200":{"description":"ok","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Widget"}}}}}`
	const both = `"requestBody":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Widget"}}}},` + response
	const spec = `{"openapi":"3.1.0","info":{"title":"Direction","version":"1.0.0"},` +
		`"paths":{"/widgets":{"post":{%s}}},` +
		`"components":{"schemas":{"Widget":{"type":"object","required":[%s],"properties":{"name":{"type":"string"},"id":{"type":"string"}}}}}}`
	for _, tc := range []struct {
		name, operation, old, next string
		breaking                   bool
	}{
		{"response schema gains a required entry", response, `"name"`, `"name","id"`, false},
		{"response schema loses a required entry", response, `"name","id"`, `"name"`, true},
		{"request schema gains a required entry", request, `"name"`, `"name","id"`, true},
		{"request schema loses a required entry", request, `"name","id"`, `"name"`, false},
		{"shared schema gains a required entry", both, `"name"`, `"name","id"`, true},
		{"shared schema loses a required entry", both, `"name","id"`, `"name"`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left, err := NewDocument([]byte(fmt.Sprintf(spec, tc.operation, tc.old)))
			require.NoError(t, err)
			right, err := NewDocument([]byte(fmt.Sprintf(spec, tc.operation, tc.next)))
			require.NoError(t, err)
			changes, err := CompareDocuments(left, right)
			require.NoError(t, err)
			require.NotNil(t, changes)
			assert.Equal(t, 1, changes.TotalChanges())
			assert.Equal(t, tc.breaking, changes.TotalBreakingChanges() > 0)
		})
	}
}

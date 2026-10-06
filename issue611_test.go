// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"fmt"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestIssue611CompareDocumentsObjectComposition(t *testing.T) {
	const object = `{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}}`
	const nullable = `{"type":["object","null"],"required":["id"],"properties":{"id":{"type":"string"}}}`
	const stringUnion = `{"type":["object","string"],"required":["id"],"properties":{"id":{"type":"string"}}}`
	const spec = `{"openapi":"3.1.0","info":{"title":"Composition","version":"1.0.0"},"paths":{},"components":{"schemas":{"Entry":%s}}}`
	for _, tc := range []struct {
		name, old, next   string
		changed, breaking bool
	}{
		{"equivalent nullable refactor", nullable, `{"oneOf":[` + object + `,{"type":"null"}]}`, false, false},
		{"array widening", object, `{"anyOf":[` + object + `,{"type":"array"}]}`, true, false},
		{"null widening", object, `{"oneOf":[` + object + `,{"type":"null"}]}`, true, false},
		{"overlapping oneOf", object, `{"oneOf":[` + object + `,{"type":"object"}]}`, true, true},
		{"wrapper constraint", object, `{"anyOf":[` + object + `,{"type":"null"}],"maxProperties":0}`, true, true},
		{"dropped string alternative", stringUnion, `{"anyOf":[` + object + `,{"type":"null"}]}`, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left, err := NewDocument([]byte(fmt.Sprintf(spec, tc.old)))
			require.NoError(t, err)
			right, err := NewDocument([]byte(fmt.Sprintf(spec, tc.next)))
			require.NoError(t, err)
			changes, err := CompareDocuments(left, right)
			require.NoError(t, err)
			assert.Equal(t, tc.changed, changes.TotalChanges() > 0)
			assert.Equal(t, tc.breaking, changes.TotalBreakingChanges() > 0)
		})
	}
}

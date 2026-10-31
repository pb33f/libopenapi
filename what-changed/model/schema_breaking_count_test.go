package model

import (
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestCompareSchemas_AllOfBreakingChangesCountedOnce(t *testing.T) {
	left := `openapi: 3.0.3
components:
  schemas:
    Root:
      allOf:
        - type: string
`
	right := strings.Replace(left, "type: string", "type: integer", 1)
	l, r := test_BuildDoc(left, right)
	changes := CompareSchemas(l.Components.Value.FindSchema("Root").Value, r.Components.Value.FindSchema("Root").Value)
	require.Equal(t, 1, changes.TotalChanges())
	require.Equal(t, 1, changes.TotalBreakingChanges())
	require.Len(t, changes.GetAllChanges(), 1)
	require.True(t, changes.GetAllChanges()[0].Breaking)
}

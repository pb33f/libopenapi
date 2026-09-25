// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT
package libopenapi

import (
	"testing"

	"github.com/pb33f/libopenapi/what-changed/model"
	"github.com/pb33f/testify/require"
)

const issue620Base = `openapi: 3.0.3
info:
  title: T
  version: 1.0.0
paths:
  /a:
    get:
      responses:
        '200':
          description: ok
        '400':
          $ref: '#/components/responses/Err'
  /b:
    get:
      responses:
        '200':
          description: ok
        '409':
          $ref: '#/components/responses/Err'
components:
  responses:
    Err:
      description: failed
`

const issue620Modified = `openapi: 3.0.3
info:
  title: T
  version: 1.0.0
paths:
  /a:
    get:
      responses:
        '200':
          description: ok
  /b:
    get:
      responses:
        '200':
          description: ok
components:
  responses:
    Err:
      description: failed
`

// Two map entries that $ref the same component share one resolved value node. Reporting a removal must
// describe each entry by its own key, and must not write that key into the shared component node.
func TestIssue620RemovedRefEntriesReportTheirOwnKeys(t *testing.T) {
	for i := 0; i < 10; i++ {
		left, err := NewDocument([]byte(issue620Base))
		require.NoError(t, err)
		right, err := NewDocument([]byte(issue620Modified))
		require.NoError(t, err)

		changes, errs := CompareDocuments(left, right)
		require.Empty(t, errs)
		require.NotNil(t, changes)

		for path, code := range map[string]string{"/a": "400", "/b": "409"} {
			pathChanges := changes.PathsChanges.PathItemsChanges[path]
			require.NotNil(t, pathChanges)
			removed := pathChanges.GetChanges.ResponsesChanges.Changes
			require.Len(t, removed, 1)
			require.Equal(t, model.ObjectRemoved, removed[0].ChangeType)
			require.Equal(t, code, removed[0].Original, "path %s reported the wrong response code", path)
		}

		leftModel, err := left.BuildV3Model()
		require.NoError(t, err)
		errResponse := leftModel.Model.GoLow().Components.Value.FindResponse("Err")
		require.NotNil(t, errResponse)
		require.Empty(t, errResponse.ValueNode.Value, "the shared component node must not be modified")
	}
}

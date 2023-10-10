// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rolodex

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/fstest"
)

func TestFilesCorrectlyListsFilesInMapFS(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"spec.yaml":                   {Data: []byte("hip")},
		"components/utils/spec.json":  {Data: []byte("hop")},
		"definitions/utils/spec.json": {Data: []byte("chip")},
		"somewhere/spec.yaml":         {Data: []byte("shop")},
	}
	found := Files(".", fsys)
	assert.Len(t, found.Files, 4)
	assert.Equal(t, string(found.FindFile("spec.yaml").Content), "hip")
	assert.Equal(t, string(found.FindFile("components/utils/spec.json").Content), "hop")

}

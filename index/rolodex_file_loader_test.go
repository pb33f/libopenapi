// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/fstest"
	"time"
)

func TestRolodexLoadsFilesCorrectly_NoErrors(t *testing.T) {
	t.Parallel()
	testFS := fstest.MapFS{
		"spec.yaml":             {Data: []byte("hip"), ModTime: time.Now()},
		"subfolder/spec1.json":  {Data: []byte("hop"), ModTime: time.Now()},
		"subfolder2/spec2.yaml": {Data: []byte("chop"), ModTime: time.Now()},
		"subfolder2/hello.jpg":  {Data: []byte("shop"), ModTime: time.Now()},
	}

	fileFS, err := NewLocalFS(".", testFS)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, fileFS.Files, 3)
	assert.Len(t, fileFS.readingErrors, 0)
}

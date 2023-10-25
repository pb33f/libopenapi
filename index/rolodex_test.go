// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/stretchr/testify/assert"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestRolodex_LocalNativeFS(t *testing.T) {

	t.Parallel()
	testFS := fstest.MapFS{
		"spec.yaml":             {Data: []byte("hip"), ModTime: time.Now()},
		"subfolder/spec1.json":  {Data: []byte("hop"), ModTime: time.Now()},
		"subfolder2/spec2.yaml": {Data: []byte("chop"), ModTime: time.Now()},
		"subfolder2/hello.jpg":  {Data: []byte("shop"), ModTime: time.Now()},
	}

	baseDir := "/tmp"

	fileFS, err := NewLocalFS(baseDir, testFS)
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS(baseDir, fileFS)

	f, rerr := rolo.Open("spec.yaml")
	assert.NoError(t, rerr)
	assert.Equal(t, "hip", f.GetContent())

}

func TestRolodex_LocalNonNativeFS(t *testing.T) {

	t.Parallel()
	testFS := fstest.MapFS{
		"spec.yaml":             {Data: []byte("hip"), ModTime: time.Now()},
		"subfolder/spec1.json":  {Data: []byte("hop"), ModTime: time.Now()},
		"subfolder2/spec2.yaml": {Data: []byte("chop"), ModTime: time.Now()},
		"subfolder2/hello.jpg":  {Data: []byte("shop"), ModTime: time.Now()},
	}

	baseDir := ""

	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS(baseDir, testFS)

	f, rerr := rolo.Open("spec.yaml")
	assert.NoError(t, rerr)

	assert.Equal(t, "hip", f.GetContent())
}

func TestRolodex_SimpleTest_OneDoc(t *testing.T) {

	baseDir := "rolodex_test_data"

	fileFS, err := NewLocalFS(baseDir, os.DirFS(baseDir))
	if err != nil {
		t.Fatal(err)
	}

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	err = rolo.IndexTheRolodex()

	assert.NoError(t, err)
	assert.Len(t, rolo.indexes, 9)

	// open components.yaml
	f, rerr := rolo.Open("components.yaml")
	assert.NoError(t, rerr)
	assert.Equal(t, "components.yaml", f.Name())

	idx, ierr := f.(*rolodexFile).Index(cf)
	assert.NoError(t, ierr)
	assert.NotNil(t, idx)
	assert.Equal(t, YAML, f.GetFileExtension())
	assert.True(t, strings.HasSuffix(f.GetFullPath(), "rolodex_test_data/components.yaml"))
	assert.Equal(t, "2023-10-12", f.ModTime().Format("2006-01-02"))
	assert.Equal(t, int64(283), f.Size())
	assert.False(t, f.IsDir())
	assert.Nil(t, f.Sys())
	assert.Equal(t, fs.FileMode(0), f.Mode())
}

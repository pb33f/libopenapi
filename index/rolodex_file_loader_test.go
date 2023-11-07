// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
)

func TestRolodexLoadsFilesCorrectly_NoErrors(t *testing.T) {
	t.Parallel()
	testFS := fstest.MapFS{
		"spec.yaml":             {Data: []byte("hip"), ModTime: time.Now()},
		"spock.yaml":            {Data: []byte("hip: : hello:  :\n:hw"), ModTime: time.Now()},
		"subfolder/spec1.json":  {Data: []byte("hop"), ModTime: time.Now()},
		"subfolder2/spec2.yaml": {Data: []byte("chop"), ModTime: time.Now()},
		"subfolder2/hello.jpg":  {Data: []byte("shop"), ModTime: time.Now()},
	}

	fileFS, err := NewLocalFS(".", testFS)
	if err != nil {
		t.Fatal(err)
	}

	files := fileFS.GetFiles()
	assert.Len(t, files, 4)
	assert.Len(t, fileFS.GetErrors(), 0)

	key, _ := filepath.Abs(filepath.Join(fileFS.baseDirectory, "spec.yaml"))

	localFile := files[key]
	assert.NotNil(t, localFile)
	assert.Nil(t, localFile.GetIndex())

	lf := localFile.(*LocalFile)
	idx, ierr := lf.Index(CreateOpenAPIIndexConfig())
	assert.NoError(t, ierr)
	assert.NotNil(t, idx)
	assert.NotNil(t, localFile.GetContent())

	d, e := localFile.GetContentAsYAMLNode()
	assert.NoError(t, e)
	assert.NotNil(t, d)
	assert.NotNil(t, localFile.GetIndex())
	assert.Equal(t, YAML, localFile.GetFileExtension())
	assert.Equal(t, key, localFile.GetFullPath())
	assert.Equal(t, "spec.yaml", lf.Name())
	assert.Equal(t, int64(3), lf.Size())
	assert.Equal(t, fs.FileMode(0), lf.Mode())
	assert.False(t, lf.IsDir())
	assert.Equal(t, time.Now().Unix(), lf.ModTime().Unix())
	assert.Nil(t, lf.Sys())
	assert.Nil(t, lf.Close())
	q, w := lf.Stat()
	assert.NotNil(t, q)
	assert.NoError(t, w)

	b, x := io.ReadAll(lf)
	assert.Len(t, b, 3)
	assert.NoError(t, x)

	assert.Equal(t, key, lf.FullPath())
	assert.Len(t, localFile.GetErrors(), 0)

	// try and reindex
	idx, ierr = lf.Index(CreateOpenAPIIndexConfig())
	assert.NoError(t, ierr)
	assert.NotNil(t, idx)

	key, _ = filepath.Abs(filepath.Join(fileFS.baseDirectory, "spock.yaml"))

	localFile = files[key]
	assert.NotNil(t, localFile)
	assert.Nil(t, localFile.GetIndex())

	lf = localFile.(*LocalFile)
	idx, ierr = lf.Index(CreateOpenAPIIndexConfig())
	assert.Error(t, ierr)
	assert.Nil(t, idx)
	assert.NotNil(t, localFile.GetContent())
	assert.Nil(t, localFile.GetIndex())

}

func TestRolodexLocalFS_NoConfig(t *testing.T) {

	lfs := &LocalFS{}
	f, e := lfs.Open("test.yaml")
	assert.Nil(t, f)
	assert.Error(t, e)
}

func TestRolodexLocalFS_NoLookup(t *testing.T) {

	cf := CreateClosedAPIIndexConfig()
	lfs := &LocalFS{indexConfig: cf}
	f, e := lfs.Open("test.yaml")
	assert.Nil(t, f)
	assert.Error(t, e)
}

func TestRolodexLocalFS_BadAbsFile(t *testing.T) {

	cf := CreateOpenAPIIndexConfig()
	lfs := &LocalFS{indexConfig: cf}
	f, e := lfs.Open("/test.yaml")
	assert.Nil(t, f)
	assert.Error(t, e)
}

func TestRolodexLocalFile_BadParse(t *testing.T) {

	lf := &LocalFile{}
	n, e := lf.GetContentAsYAMLNode()
	assert.Nil(t, n)
	assert.Error(t, e)
	assert.Equal(t, "no data to parse for file: ", e.Error())
}

func TestRolodexLocalFile_NoIndexRoot(t *testing.T) {

	lf := &LocalFile{data: []byte("burders"), index: &SpecIndex{}}
	n, e := lf.GetContentAsYAMLNode()
	assert.NotNil(t, n)
	assert.NoError(t, e)

}

func TestRolodexLocalFS_NoBaseRelative(t *testing.T) {

	lfs := &LocalFS{}
	f, e := lfs.extractFile("test.jpg")
	assert.Nil(t, f)
	assert.NoError(t, e)
}

func TestRolodexLocalFile_IndexSingleFile(t *testing.T) {

	testFS := fstest.MapFS{
		"spec.yaml":  {Data: []byte("hip"), ModTime: time.Now()},
		"spock.yaml": {Data: []byte("hop"), ModTime: time.Now()},
		"i-am-a-dir": {Mode: fs.FileMode(fs.ModeDir), ModTime: time.Now()},
	}

	fileFS, _ := NewLocalFS("spec.yaml", testFS)
	files := fileFS.GetFiles()
	assert.Len(t, files, 1)

}

func TestRolodexLocalFile_TestFilters(t *testing.T) {

	testFS := fstest.MapFS{
		"spec.yaml":  {Data: []byte("hip"), ModTime: time.Now()},
		"spock.yaml": {Data: []byte("pip"), ModTime: time.Now()},
		"jam.jpg":    {Data: []byte("sip"), ModTime: time.Now()},
	}

	fileFS, _ := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: ".",
		FileFilters:   []string{"spec.yaml", "spock.yaml", "jam.jpg"},
		DirFS:         testFS,
	})
	files := fileFS.GetFiles()
	assert.Len(t, files, 2)

}

func TestRolodexLocalFile_TestBadFS(t *testing.T) {

	testFS := test_badfs{}

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: ".",
		DirFS:         &testFS,
	})
	assert.Error(t, err)
	assert.Nil(t, fileFS)

}

func TestNewRolodexLocalFile_BadOffset(t *testing.T) {

	lf := &LocalFile{offset: -1}
	z, y := io.ReadAll(lf)
	assert.Len(t, z, 0)
	assert.Error(t, y)
}

func TestRecursiveLocalFile_IndexFail(t *testing.T) {

	pup := []byte("I:\n miss you fox, you're: my good boy:")

	var myPuppy yaml.Node
	_ = yaml.Unmarshal(pup, &myPuppy)

	_ = os.WriteFile("fox.yaml", pup, 0o664)
	defer os.Remove("fox.yaml")

	// create a new config that allows local and remote to be mixed up.
	cf := CreateOpenAPIIndexConfig()
	cf.AvoidBuildIndex = true

	// create a new rolodex
	rolo := NewRolodex(cf)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&myPuppy)

	// configure the local filesystem.
	fsCfg := LocalFSConfig{
		IndexConfig: cf,
	}

	// create a new local filesystem.
	fileFS, err := NewLocalFSWithConfig(&fsCfg)
	assert.NoError(t, err)

	rolo.AddLocalFS(cf.BasePath, fileFS)
	rErr := rolo.IndexTheRolodex()

	assert.NoError(t, rErr)

	fox, fErr := rolo.Open("fox.yaml")
	assert.NoError(t, fErr)
	assert.NotNil(t, fox)
	assert.Len(t, fox.GetErrors(), 1)
	assert.Equal(t, "unable to parse specification: yaml: line 2: mapping values are not allowed in this context", fox.GetErrors()[0].Error())

}

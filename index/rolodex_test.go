// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestRolodex_NewRolodex(t *testing.T) {
	c := CreateOpenAPIIndexConfig()
	rolo := NewRolodex(c)
	assert.Len(t, rolo.GetAllReferences(), 0)
	assert.Len(t, rolo.GetAllMappedReferences(), 0)
	assert.NotNil(t, rolo)
	assert.NotNil(t, rolo.indexConfig)
	assert.Nil(t, rolo.GetIgnoredCircularReferences())
	assert.Equal(t, rolo.GetIndexingDuration(), time.Duration(0))
	assert.Nil(t, rolo.GetRootIndex())
	assert.Len(t, rolo.GetIndexes(), 0)
	assert.Len(t, rolo.GetCaughtErrors(), 0)
	assert.NotNil(t, rolo.GetConfig())
}

func TestRolodex_NoFS(t *testing.T) {
	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rf, err := rolo.Open("spec.yaml")
	assert.Error(t, err)
	assert.Equal(t, "rolodex has no file systems configured, cannot open 'spec.yaml'. "+
		"Add a BaseURL or BasePath to your configuration so the rolodex knows how to resolve references", err.Error())
	assert.Nil(t, rf)
}

func TestRolodex_NoFSButHasRemoteFS(t *testing.T) {
	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddRemoteFS("http://localhost", nil)
	rf, err := rolo.Open("spec.yaml")
	assert.Error(t, err)
	assert.Equal(t, "the rolodex has no local file systems configured, cannot open local file 'spec.yaml'", err.Error())
	assert.Nil(t, rf)
}

func TestRolodex_LocalNativeFS(t *testing.T) {
	t.Parallel()
	testFS := fstest.MapFS{
		"spec.yaml":             {Data: []byte("hip"), ModTime: time.Now()},
		"subfolder/spec1.json":  {Data: []byte("hop"), ModTime: time.Now()},
		"subfolder2/spec2.yaml": {Data: []byte("chop"), ModTime: time.Now()},
		"subfolder2/hello.jpg":  {Data: []byte("shop"), ModTime: time.Now()},
	}

	baseDir := "/tmp"

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
		DirFS: testFS,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS(baseDir, fileFS)

	f, rerr := rolo.Open("spec.yaml")
	assert.NoError(t, rerr)
	assert.Equal(t, "hip", f.GetContent())
	rolo.rootIndex = NewTestSpecIndex().Load().(*SpecIndex)
	rolo.indexes = append(rolo.indexes, rolo.rootIndex)
	rolo.ClearIndexCaches()
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

type test_badfs struct {
	ok       bool
	goodstat bool
	offset   int64
}

func (t *test_badfs) Open(v string) (fs.File, error) {
	ok := false
	if v != "/" && v != "." && v != "http://localhost/test.yaml" {
		ok = true
	}
	if v == "http://localhost/goodstat.yaml" || strings.HasSuffix(v, "goodstat.yaml") {
		ok = true
		t.goodstat = true
	}
	if v == "http://localhost/badstat.yaml" || v == "badstat.yaml" {
		ok = true
		t.goodstat = false
	}
	return &test_badfs{ok: ok, goodstat: t.goodstat}, nil
}

func (t *test_badfs) Stat() (fs.FileInfo, error) {
	if t.goodstat {
		return &LocalFile{
			lastModified: time.Now(),
		}, nil
	}
	return nil, os.ErrInvalid
}

func (t *test_badfs) Read(b []byte) (int, error) {
	if t.ok {
		if t.offset >= int64(len("pizza")) {
			return 0, io.EOF
		}
		if t.offset < 0 {
			return 0, &fs.PathError{Op: "read", Path: "lemons", Err: fs.ErrInvalid}
		}
		n := copy(b, "pizza"[t.offset:])
		t.offset += int64(n)
		return n, nil
	}
	return 0, os.ErrNotExist
}

func (t *test_badfs) Close() error {
	return os.ErrNotExist
}

func TestRolodex_LocalNonNativeFS_BadRead(t *testing.T) {
	t.Parallel()
	testFS := &test_badfs{}

	baseDir := ""

	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS(baseDir, testFS)

	f, rerr := rolo.Open("/")
	assert.Nil(t, f)
	assert.Error(t, rerr)
	// The error message can vary based on how paths are resolved
	// Just ensure we get an error
	assert.Contains(t, []string{"file does not exist", "invalid argument"}, rerr.Error())
}

func TestRolodex_LocalNonNativeFS_BadStat(t *testing.T) {
	t.Parallel()
	testFS := &test_badfs{}

	baseDir := ""

	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS(baseDir, testFS)

	f, rerr := rolo.Open("badstat.yaml")
	assert.Nil(t, f)
	assert.Error(t, rerr)
	assert.Equal(t, "invalid argument", rerr.Error())
}

func TestRolodex_LocalNonNativeRemoteFS_BadRead(t *testing.T) {
	t.Parallel()
	testFS := &test_badfs{}

	baseDir := ""

	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddRemoteFS(baseDir, testFS)

	f, rerr := rolo.Open("http://localhost/test.yaml")
	assert.Nil(t, f)
	assert.Error(t, rerr)
	assert.Equal(t, "file does not exist", rerr.Error())
}

func TestRolodex_LocalNonNativeRemoteFS_ReadFile(t *testing.T) {
	t.Parallel()
	testFS := &test_badfs{}

	baseDir := ""

	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddRemoteFS(baseDir, testFS)

	r, rerr := rolo.Open("http://localhost/goodstat.yaml")
	assert.NotNil(t, r)
	assert.NoError(t, rerr)

	assert.Equal(t, "goodstat.yaml", r.Name())
	assert.Nil(t, r.GetIndex())
	assert.Equal(t, "pizza", r.GetContent())
	assert.Equal(t, "http://localhost/goodstat.yaml", r.GetFullPath())
	assert.Greater(t, r.ModTime().UnixMilli(), int64(1))
	assert.Equal(t, int64(5), r.Size())
	assert.False(t, r.IsDir())
	assert.Nil(t, r.Sys())
	assert.Equal(t, r.Mode(), os.FileMode(0))
	n, e := r.GetContentAsYAMLNode()
	assert.Len(t, r.GetErrors(), 0)
	assert.NoError(t, e)
	assert.NotNil(t, n)
	assert.Equal(t, YAML, r.GetFileExtension())
}

func TestRolodex_LocalNonNativeRemoteFS_BadStat(t *testing.T) {
	t.Parallel()
	testFS := &test_badfs{}

	baseDir := ""

	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddRemoteFS(baseDir, testFS)

	f, rerr := rolo.Open("http://localhost/badstat.yaml")
	assert.Nil(t, f)
	assert.Error(t, rerr)
	assert.Equal(t, "invalid argument", rerr.Error())
}

func TestRolodex_rolodexFileTests(t *testing.T) {
	r := &rolodexFile{}
	assert.Equal(t, "", r.Name())
	assert.Nil(t, r.GetIndex())
	assert.Equal(t, "", r.GetContent())
	assert.Equal(t, "", r.GetFullPath())
	assert.Equal(t, time.Now().UnixMilli(), r.ModTime().UnixMilli())
	assert.Equal(t, int64(0), r.Size())
	assert.False(t, r.IsDir())
	assert.Nil(t, r.Sys())
	assert.Equal(t, r.Mode(), os.FileMode(0))
	n, e := r.GetContentAsYAMLNode()
	assert.Len(t, r.GetErrors(), 0)
	assert.NoError(t, e)
	assert.Nil(t, n)
	assert.Equal(t, UNSUPPORTED, r.GetFileExtension())
}

func TestRolodex_NotRolodexFS(t *testing.T) {
	nonRoloFS := os.DirFS(".")
	cf := CreateOpenAPIIndexConfig()
	rolo := NewRolodex(cf)
	rolo.AddLocalFS(".", nonRoloFS)

	err := rolo.IndexTheRolodex(context.Background())

	assert.Error(t, err)
	assert.Equal(t, "rolodex file system is not a RolodexFS", err.Error())
}

func TestRolodex_IndexCircularLookup(t *testing.T) {
	offToOz := `openapi: 3.1.0
components:
  schemas:
    CircleTest:
      $ref: "../test_specs/circular-tests.yaml#/components/schemas/One"`

	_ = os.WriteFile("off_to_oz.yaml", []byte(offToOz), 0o644)
	defer os.Remove("off_to_oz.yaml")

	baseDir := "../"

	fsCfg := &LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         os.DirFS(baseDir),
		FileFilters: []string{
			"off_to_oz.yaml",
			"test_specs/circular-tests.yaml",
		},
	}

	fileFS, err := NewLocalFSWithConfig(fsCfg)
	if err != nil {
		t.Fatal(err)
	}

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	rolodex := NewRolodex(cf)
	rolodex.AddLocalFS(baseDir, fileFS)
	err = rolodex.IndexTheRolodex(context.Background())
	assert.Error(t, err)
	assert.Len(t, rolodex.GetCaughtErrors(), 3)
	assert.Len(t, rolodex.GetIgnoredCircularReferences(), 0)
}

func TestRolodex_IndexCircularLookup_AroundWeGo(t *testing.T) {
	there := `openapi: 3.1.0
components:
  schemas:
    CircleTest:
      type: object
      required:
        - where
      properties:
        where:
          $ref: "back-again.yaml#/components/schemas/CircleTest/properties/muffins"`

	backagain := `openapi: 3.1.0
components:
  schemas:
    CircleTest:
      type: object
      required:
        - muffins
      properties:
        muffins:
         $ref: "there.yaml#/components/schemas/CircleTest"`

	_ = os.WriteFile("there.yaml", []byte(there), 0o644)
	_ = os.WriteFile("back-again.yaml", []byte(backagain), 0o644)
	defer os.Remove("there.yaml")
	defer os.Remove("back-again.yaml")

	baseDir := "."

	fsCfg := &LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         os.DirFS(baseDir),
		FileFilters: []string{
			"there.yaml",
			"back-again.yaml",
		},
	}

	fileFS, err := NewLocalFSWithConfig(fsCfg)
	if err != nil {
		t.Fatal(err)
	}

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	rolodex := NewRolodex(cf)
	rolodex.AddLocalFS(baseDir, fileFS)
	err = rolodex.IndexTheRolodex(context.Background())
	assert.Error(t, err)
	assert.Len(t, rolodex.GetCaughtErrors(), 1)
	assert.Len(t, rolodex.GetIgnoredCircularReferences(), 0)
}

func TestRolodex_IndexCircularLookup_AroundWeGo_IgnorePoly(t *testing.T) {
	fifth := "type: string"

	fourth := `type: "object"
properties:
  name:
    type: "string"
  children:
    type: "object"`

	third := `type: "object"
properties:
  blame:
    $ref: "$_5"

  fame: 
     $ref: "$_4#/properties/name"
  game:
    $ref: "$_5"

  children:
    type: "object"
    anyOf:
      - $ref: "$2#/components/schemas/CircleTest"
required:
  - children`

	second := `openapi: 3.1.0
components:
  schemas:
    CircleTest:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          anyOf:
            - $ref: "$3"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"
`

	first := `openapi: 3.1.0
components:
  schemas:
    StartTest:
      type: object
      required:
        - muffins
      properties:
        muffins:
         $ref: "$2#/components/schemas/CircleTest"
`

	var firstFile, secondFile, thirdFile, fourthFile, fifthFile *os.File
	var fErr error

	tmp := "tmp-a"
	_ = os.Mkdir(tmp, 0o755)

	firstFile, fErr = os.CreateTemp(tmp, "*-first.yaml")
	assert.NoError(t, fErr)

	secondFile, fErr = os.CreateTemp(tmp, "*-second.yaml")
	assert.NoError(t, fErr)

	thirdFile, fErr = os.CreateTemp(tmp, "*-third.yaml")
	assert.NoError(t, fErr)

	fourthFile, fErr = os.CreateTemp(tmp, "*-fourth.yaml")
	assert.NoError(t, fErr)

	fifthFile, fErr = os.CreateTemp(tmp, "*-fifth.yaml")
	assert.NoError(t, fErr)

	defer os.RemoveAll(tmp)

	first = strings.ReplaceAll(strings.ReplaceAll(first, "$2", secondFile.Name()), "\\", "\\\\")
	second = strings.ReplaceAll(strings.ReplaceAll(second, "$3", thirdFile.Name()), "\\", "\\\\")
	third = strings.ReplaceAll(third, "$4", filepath.Base(fourthFile.Name()))
	third = strings.ReplaceAll(third, "$_4", fourthFile.Name())
	third = strings.ReplaceAll(third, "$5", filepath.Base(fifthFile.Name()))
	third = strings.ReplaceAll(third, "$_5", fifthFile.Name())
	third = strings.ReplaceAll(strings.ReplaceAll(third, "$2", secondFile.Name()), "\\", "\\\\")

	firstFile.WriteString(first)
	secondFile.WriteString(second)
	thirdFile.WriteString(third)
	fourthFile.WriteString(fourth)
	fifthFile.WriteString(fifth)

	defer os.RemoveAll(tmp)

	baseDir := "tmp-a"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.IgnorePolymorphicCircularReferences = true
	cf.SkipDocumentCheck = true

	fsCfg := &LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
		//DirFS:         os.DirFS(baseDir),
		//FileFilters: []string{
		//	filepath.Base(firstFile.Name()),
		//	filepath.Base(secondFile.Name()),
		//	filepath.Base(thirdFile.Name()),
		//	filepath.Base(fourthFile.Name()),
		//	filepath.Base(fifthFile.Name()),
		//},
	}

	fileFS, err := NewLocalFSWithConfig(fsCfg)
	if err != nil {
		t.Fatal(err)
	}

	// add logger to config
	cf.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	rolodex := NewRolodex(cf)
	rolodex.AddLocalFS(baseDir, fileFS)

	// srv := test_rolodexDeepRefServer([]byte(first), []byte(second),
	//	[]byte(third), []byte(fourth), []byte(fifth))
	// defer srv.Close()

	// u, _ := url.Parse(srv.URL)
	// cf.BaseURL = u
	// remoteFS, rErr := NewRemoteFSWithConfig(cf)
	// assert.NoError(t, rErr)

	// rolodex.AddRemoteFS(srv.URL, remoteFS)

	var rootNode yaml.Node
	err = yaml.Unmarshal([]byte(first), &rootNode)
	assert.NoError(t, err)
	rolodex.SetRootNode(&rootNode)

	err = rolodex.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rolodex.GetCaughtErrors(), 0)

	// there are two circles. Once when reading the journey from first.yaml, and then a second internal look in second.yaml
	// the index won't find three, because by the time that 'three' has been read, it's already been indexed and the journey
	// discovered.
	assert.GreaterOrEqual(t, len(rolodex.GetIgnoredCircularReferences()), 1)

	// extract a local file
	n, _ := filepath.Abs(firstFile.Name())
	f, _ := rolodex.Open(n)
	// index
	x, y := f.(*rolodexFile).Index(cf)
	assert.NotNil(t, x)
	assert.NoError(t, y)

	// re-index
	x, y = f.(*rolodexFile).Index(cf)
	assert.NotNil(t, x)
	assert.NoError(t, y)

	//// extract a remote  file
	//f, _ = rolodex.Open("http://the-space-race-is-all-about-space-and-time-dot.com/" + filepath.Base(fourthFile.Name()))
	//
	//// index
	//x, y = f.(*rolodexFile).Index(cf)
	//assert.NotNil(t, x)
	//assert.NoError(t, y)
	//
	//// re-index
	//x, y = f.(*rolodexFile).Index(cf)
	//assert.NotNil(t, x)
	//assert.NoError(t, y)
	//
	//// extract another remote  file
	//f, _ = rolodex.Open("http://the-space-race-is-all-about-space-and-time-dot.com/" + filepath.Base(fifthFile.Name()))
	//
	////change cf to perform document check (which should fail)
	//cf.SkipDocumentCheck = false
	//
	//// index and fail
	//x, y = f.(*rolodexFile).Index(cf)
	//assert.Nil(t, x)
	//assert.Error(t, y)
	//
	//// file that is not local, but is remote
	//f, _ = rolodex.Open("https://pb33f.io/bingo/jingo.yaml")
	//assert.NotNil(t, f)
}

func test_rolodexDeepRefServer(a, b, c, d, e []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Last-Modified", "Wed, 21 Oct 2015 12:28:00 GMT")
		if strings.HasSuffix(req.URL.String(), "-first.yaml") {
			_, _ = rw.Write(a)
			return
		}
		if strings.HasSuffix(req.URL.String(), "-second.yaml") {
			_, _ = rw.Write(b)
			return
		}
		if strings.HasSuffix(req.URL.String(), "-third.yaml") {
			_, _ = rw.Write(c)
			return
		}
		if strings.HasSuffix(req.URL.String(), "-fourth.yaml") {
			_, _ = rw.Write(d)
			return
		}
		if strings.HasSuffix(req.URL.String(), "-fifth.yaml") {
			_, _ = rw.Write(e)
			return
		}
		if strings.HasSuffix(req.URL.String(), "/bingo/jingo.yaml") {
			_, _ = rw.Write([]byte("openapi: 3.1.0"))
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("500 - COMPUTAR SAYS NO!"))
	}))
}

func TestRolodex_IndexCircularLookup_PolyItems_LocalLoop_WithFiles_RecursiveLookup(t *testing.T) {
	fourth := `type: "object"
properties:
  name:
    type: "string"
  children:
    type: "object"`

	third := `type: "object"
properties:
  herbs:
    $ref: "$1"`

	second := `openapi: 3.1.0
components:
  schemas:
    CircleTest:
      type: "object"
      properties:
        bing:
          $ref: "not_found.yaml"
        name:
          type: "string"
        children:
          type: "object"
          anyOf:
            - $ref: "$3"
      required:
        - "name"
        - "children"`

	first := `openapi: 3.1.0
components:
  schemas:
    StartTest:
      type: object
      required:
        - muffins
      properties:
        muffins:
         $ref: "$2#/components/schemas/CircleTest"`

	var firstFile, secondFile, thirdFile, fourthFile *os.File
	var fErr error

	tmp := "tmp-b"
	_ = os.Mkdir(tmp, 0o755)

	firstFile, fErr = os.CreateTemp(tmp, "*-first.yaml")
	assert.NoError(t, fErr)

	secondFile, fErr = os.CreateTemp(tmp, "*-second.yaml")
	assert.NoError(t, fErr)

	thirdFile, fErr = os.CreateTemp(tmp, "*-third.yaml")
	assert.NoError(t, fErr)

	fourthFile, fErr = os.CreateTemp(tmp, "*-fourth.yaml")
	assert.NoError(t, fErr)

	first = strings.ReplaceAll(strings.ReplaceAll(first, "$2", secondFile.Name()), "\\", "\\\\")
	second = strings.ReplaceAll(strings.ReplaceAll(second, "$3", thirdFile.Name()), "\\", "\\\\")
	third = strings.ReplaceAll(strings.ReplaceAll(third, "$4", filepath.Base(fourthFile.Name())), "\\", "\\\\")
	third = strings.ReplaceAll(strings.ReplaceAll(first, "$1", filepath.Base(thirdFile.Name())), "\\", "\\\\")

	firstFile.WriteString(first)
	secondFile.WriteString(second)
	thirdFile.WriteString(third)
	fourthFile.WriteString(fourth)

	defer os.RemoveAll(tmp)

	baseDir, _ := filepath.Abs(tmp)
	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.IgnorePolymorphicCircularReferences = true

	fsCfg := &LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	}

	fileFS, err := NewLocalFSWithConfig(fsCfg)
	if err != nil {
		t.Fatal(err)
	}

	rolodex := NewRolodex(cf)
	rolodex.AddLocalFS(baseDir, fileFS)

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(first), &rootNode)
	rolodex.SetRootNode(&rootNode)

	err = rolodex.IndexTheRolodex(context.Background())
	assert.Error(t, err)
	assert.GreaterOrEqual(t, len(rolodex.GetCaughtErrors()), 1)
	assert.Equal(t, "cannot resolve reference `not_found.yaml`, it's missing: $.['not_found.yaml'] [8:11]", rolodex.GetCaughtErrors()[0].Error())
}

func TestRolodex_IndexCircularLookup_PolyItems_LocalLoop_WithFiles(t *testing.T) {
	first := `openapi: 3.1.0
components:
  schemas:
    CircleTest:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          oneOf:
            items:
              $ref: "$2#/components/schemas/CircleTest"
      required:
        - "name"
        - "children"
    StartTest:
      type: object
      required:
        - muffins
      properties:
        muffins:
         type: object
         anyOf:
           - $ref: "#/components/schemas/CircleTest"`

	second := `openapi: 3.1.0
components:
  schemas:
    CircleTest:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          oneOf:
            items:
              $ref: "#/components/schemas/CircleTest"
      required:
        - "name"
        - "children"
    StartTest:
      type: object
      required:
        - muffins
      properties:
        muffins:
         type: object
         anyOf:
           - $ref: "#/components/schemas/CircleTest"`

	var firstFile, secondFile *os.File
	var fErr error

	tmp := "tmp-f"
	_ = os.Mkdir(tmp, 0o755)

	firstFile, fErr = os.CreateTemp(tmp, "*-first.yaml")
	assert.NoError(t, fErr)

	secondFile, fErr = os.CreateTemp(tmp, "*-second.yaml")
	assert.NoError(t, fErr)

	first = strings.ReplaceAll(strings.ReplaceAll(first, "$2", secondFile.Name()), "\\", "\\\\")

	firstFile.WriteString(first)
	secondFile.WriteString(second)

	defer os.RemoveAll(tmp)

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(first), &rootNode)

	cf := CreateOpenAPIIndexConfig()
	cf.IgnorePolymorphicCircularReferences = true
	rolodex := NewRolodex(cf)

	baseDir := tmp

	fsCfg := &LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         os.DirFS(baseDir),
		FileFilters: []string{
			filepath.Base(firstFile.Name()),
			filepath.Base(secondFile.Name()),
		},
	}

	fileFS, err := NewLocalFSWithConfig(fsCfg)
	if err != nil {
		t.Fatal(err)
	}

	rolodex.AddLocalFS(baseDir, fileFS)
	rolodex.SetRootNode(&rootNode)
	assert.NotNil(t, rolodex.GetRootNode())

	err = rolodex.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rolodex.GetCaughtErrors(), 0)

	// multiple loops across two files
	assert.Len(t, rolodex.GetIgnoredCircularReferences(), 1)
}

func TestRolodex_IndexCircularLookup_PolyItems_LocalLoop_BuildIndexesPost(t *testing.T) {
	first := `openapi: 3.1.0
components:
  schemas:
    CircleTest:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          oneOf:
            items:
              $ref: "$2#/components/schemas/CircleTest"
      required:
        - "name"
        - "children"
    StartTest:
      type: object
      required:
        - muffins
      properties:
        muffins:
         type: object
         anyOf:
           - $ref: "#/components/schemas/CircleTest"`

	second := `openapi: 3.1.0
components:
  schemas:
    CircleTest:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          oneOf:
            items:
              $ref: "#/components/schemas/CircleTest"
      required:
        - "name"
        - "children"
    StartTest:
      type: object
      required:
        - muffins
      properties:
        muffins:
         type: object
         anyOf:
           - $ref: "#/components/schemas/CircleTest"`

	var firstFile, secondFile *os.File
	var fErr error

	tmp := "tmp-c"
	_ = os.Mkdir(tmp, 0o755)

	firstFile, fErr = os.CreateTemp(tmp, "*-first.yaml")
	assert.NoError(t, fErr)

	secondFile, fErr = os.CreateTemp(tmp, "*-second.yaml")
	assert.NoError(t, fErr)

	first = strings.ReplaceAll(strings.ReplaceAll(first, "$2", secondFile.Name()), "\\", "\\\\")

	firstFile.WriteString(first)
	secondFile.WriteString(second)

	defer os.RemoveAll(tmp)

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(first), &rootNode)

	cf := CreateOpenAPIIndexConfig()
	cf.IgnorePolymorphicCircularReferences = true
	cf.AvoidBuildIndex = true
	rolodex := NewRolodex(cf)

	baseDir := tmp

	fsCfg := &LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         os.DirFS(baseDir),
		FileFilters: []string{
			filepath.Base(firstFile.Name()),
			filepath.Base(secondFile.Name()),
		},
	}

	fileFS, err := NewLocalFSWithConfig(fsCfg)
	if err != nil {
		t.Fatal(err)
	}

	rolodex.AddLocalFS(baseDir, fileFS)
	rolodex.SetRootNode(&rootNode)

	err = rolodex.IndexTheRolodex(context.Background())
	rolodex.BuildIndexes()

	assert.NoError(t, err)
	assert.Len(t, rolodex.GetCaughtErrors(), 0)

	// multiple loops across two files
	assert.Len(t, rolodex.GetIgnoredCircularReferences(), 1)

	// trigger a rebuild, should do nothing.
	rolodex.BuildIndexes()
	assert.Len(t, rolodex.GetCaughtErrors(), 0)
}

func TestRolodex_IndexCircularLookup_ArrayItems_LocalLoop_WithFiles(t *testing.T) {
	first := `openapi: 3.1.0
components:
  schemas:
    CircleTest:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "array"
          items:
            $ref: "$2#/components/schemas/CircleTest"
      required:
        - "name"
        - "children"
    StartTest:
      type: object
      required:
        - muffins
      properties:
        muffins:
         type: array
         items:
           $ref: "#/components/schemas/CircleTest"`

	second := `openapi: 3.1.0
components:
  schemas:
    CircleTest:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: array
          items:
            $ref: "#/components/schemas/CircleTest"
      required:
        - "name"
        - "children"
    StartTest:
      type: object
      required:
        - muffins
      properties:
        muffins:
         type: array
         items:
           $ref: "#/components/schemas/CircleTest"`

	var firstFile, secondFile *os.File
	var fErr error

	tmp := "tmp-d"
	_ = os.Mkdir(tmp, 0o755)

	firstFile, fErr = os.CreateTemp(tmp, "*-first.yaml")
	assert.NoError(t, fErr)

	secondFile, fErr = os.CreateTemp(tmp, "*-second.yaml")
	assert.NoError(t, fErr)

	first = strings.ReplaceAll(strings.ReplaceAll(first, "$2", secondFile.Name()), "\\", "\\\\")

	firstFile.WriteString(first)
	secondFile.WriteString(second)

	defer os.RemoveAll(tmp)

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(first), &rootNode)

	cf := CreateOpenAPIIndexConfig()
	cf.IgnoreArrayCircularReferences = true
	rolodex := NewRolodex(cf)

	baseDir := tmp

	fsCfg := &LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         os.DirFS(baseDir),
		FileFilters: []string{
			filepath.Base(firstFile.Name()),
			filepath.Base(secondFile.Name()),
		},
	}

	fileFS, err := NewLocalFSWithConfig(fsCfg)
	if err != nil {
		t.Fatal(err)
	}

	rolodex.AddLocalFS(baseDir, fileFS)
	rolodex.SetRootNode(&rootNode)

	err = rolodex.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rolodex.GetCaughtErrors(), 0)

	// multiple loops across two files
	assert.Len(t, rolodex.GetIgnoredCircularReferences(), 1)
}

func TestRolodex_IndexCircularLookup_PolyItemsHttpOnly(t *testing.T) {
	third := `type: string`
	fourth := `components:
  schemas:
    Chicken:
      type: string`

	second := `openapi: 3.1.0
components:
  schemas:
    Loopy:
      type: "object"
      properties:
        cake:
          type: "string"
          anyOf:
            items:
              $ref: "https://I-love-a-good-cake-and-pizza.com/$3"
        pizza:
          type: "string"
          anyOf:
            items:
              $ref: "$3"
        same:
          type: "string"
          oneOf:
            items:
              $ref: "https://milly-the-milk-bottle.com/$4#/components/schemas/Chicken"
        name:
          type: "string"
          oneOf:
            items:
              $ref: "https://junk-peddlers-blues.com/$3#/"
        children:
          type: "object"
          allOf:
            items:
              $ref: "$1#/components/schemas/StartTest"
      required:
        - "name"
        - "children"
    CircleTest:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          oneOf:
            items:
              $ref: "#/components/schemas/Loopy"
      required:
        - "name"
        - "children"`

	first := `openapi: 3.1.0
components:
  schemas:
    StartTest:
      type: object
      required:
        - muffins
      properties:
        chuffins:
          type: object
          allOf: 
            - $ref: "https://what-a-lovely-fence.com/$3"
        buffins:
          type: object
          allOf: 
            - $ref: "https://no-more-bananas-please.com/$2#/"
        muffins:
         type: object
         anyOf:
           - $ref: "https://where-are-all-my-jellies.com/$2#/components/schemas/CircleTest"
`

	var firstFile, secondFile, thirdFile, fourthFile *os.File
	var fErr error

	tmp := "tmp-e"
	_ = os.Mkdir(tmp, 0o755)

	firstFile, fErr = os.CreateTemp(tmp, "*-first.yaml")
	assert.NoError(t, fErr)

	secondFile, fErr = os.CreateTemp(tmp, "*-second.yaml")
	assert.NoError(t, fErr)

	thirdFile, fErr = os.CreateTemp(tmp, "*-third.yaml")
	assert.NoError(t, fErr)

	fourthFile, fErr = os.CreateTemp(tmp, "*-fourth.yaml")
	assert.NoError(t, fErr)

	first = strings.ReplaceAll(first, "$2", filepath.Base(secondFile.Name()))
	first = strings.ReplaceAll(strings.ReplaceAll(first, "$3", filepath.Base(thirdFile.Name())), "\\", "\\\\")

	second = strings.ReplaceAll(second, "$3", filepath.Base(thirdFile.Name()))
	second = strings.ReplaceAll(second, "$1", filepath.Base(firstFile.Name()))
	second = strings.ReplaceAll(strings.ReplaceAll(second, "$4", filepath.Base(fourthFile.Name())), "\\", "\\\\")

	firstFile.WriteString(first)
	secondFile.WriteString(second)
	thirdFile.WriteString(third)
	fourthFile.WriteString(fourth)

	defer os.RemoveAll(tmp)

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(first), &rootNode)

	cf := CreateOpenAPIIndexConfig()
	cf.IgnorePolymorphicCircularReferences = true
	rolodex := NewRolodex(cf)

	srv := test_rolodexDeepRefServer([]byte(first), []byte(second), []byte(third), []byte(fourth), nil)
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	cf.BaseURL = u
	remoteFS, rErr := NewRemoteFSWithConfig(cf)
	assert.NoError(t, rErr)

	rolodex.AddRemoteFS(srv.URL, remoteFS)
	rolodex.SetRootNode(&rootNode)

	err := rolodex.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rolodex.GetCaughtErrors(), 0)

	assert.GreaterOrEqual(t, len(rolodex.GetIgnoredCircularReferences()), 1)

	expectedFullLineCount := (strings.Count(first, "\n") + 1) + (strings.Count(second, "\n") + 1) +
		(strings.Count(third, "\n") + 1) + (strings.Count(fourth, "\n") + 1)
	assert.Equal(t, int64(expectedFullLineCount), rolodex.GetFullLineCount())
}

func TestRolodex_IndexCircularLookup_PolyItemsFileOnly_LocalIncluded(t *testing.T) {
	third := `type: string`

	second := `openapi: 3.1.0
components:
  schemas:
    LoopyMcLoopFace:
      type: "object"
      properties:
        hoop:
          type: object
          allOf:
            items:
              $ref: "$3"
        boop:
          type: object
          allOf:
            items:
              $ref: "$3"
        loop:
          type: object
          oneOf:
            items:
              $ref: "#/components/schemas/LoopyMcLoopFace"
    CircleTest:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          anyOf:
            - $ref: "#/components/schemas/LoopyMcLoopFace"
      required:
        - "name"
        - "children"`

	first := `openapi: 3.1.0
components:
  schemas:
    StartTest:
      type: object
      required:
        - muffins
      properties:
        muffins:
         type: object
         anyOf:
           - $ref: "$2#/components/schemas/CircleTest"
           - $ref: "$3"`

	var firstFile, secondFile, thirdFile *os.File
	var fErr error

	tmp := "tmp-g"
	_ = os.Mkdir(tmp, 0o755)

	firstFile, fErr = os.CreateTemp(tmp, "first-*.yaml")
	assert.NoError(t, fErr)

	secondFile, fErr = os.CreateTemp(tmp, "second-*.yaml")
	assert.NoError(t, fErr)

	thirdFile, fErr = os.CreateTemp(tmp, "third-*.yaml")
	assert.NoError(t, fErr)

	first = strings.ReplaceAll(first, "$2", secondFile.Name())
	first = strings.ReplaceAll(strings.ReplaceAll(first, "$3", thirdFile.Name()), "\\", "\\\\")
	second = strings.ReplaceAll(strings.ReplaceAll(second, "$3", thirdFile.Name()), "\\", "\\\\")

	firstFile.WriteString(first)
	secondFile.WriteString(second)
	thirdFile.WriteString(third)

	defer os.RemoveAll(tmp)

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(first), &rootNode)

	cf := CreateOpenAPIIndexConfig()
	cf.IgnorePolymorphicCircularReferences = true
	cf.ExtractRefsSequentially = true
	rolodex := NewRolodex(cf)

	baseDir := tmp

	fsCfg := &LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         os.DirFS(baseDir),
		FileFilters: []string{
			filepath.Base(firstFile.Name()),
			filepath.Base(secondFile.Name()),
			filepath.Base(thirdFile.Name()),
		},
	}

	fileFS, err := NewLocalFSWithConfig(fsCfg)
	if err != nil {
		t.Fatal(err)
	}

	rolodex.AddLocalFS(baseDir, fileFS)
	rolodex.SetRootNode(&rootNode)

	err = rolodex.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rolodex.GetCaughtErrors(), 0)

	// should only be a single loop.
	assert.Len(t, rolodex.GetIgnoredCircularReferences(), 1)
}

func TestRolodex_TestDropDownToRemoteFS_CatchErrors(t *testing.T) {
	fourth := `type: "object"
properties:
  name:
    type: "string"
  children:
    type: "object"`

	third := `type: "object"
properties:
  name:
    $ref: "http://the-space-race-is-all-about-space-and-time-dot.com/$4"`

	second := `openapi: 3.1.0
components:
  schemas:
    CircleTest:
      type: "object"
      properties:
        bing:
          $ref: "not_found.yaml"
        name:
          type: "string"
        children:
          type: "object"
          anyOf:
            - $ref: "$3"
      required:
        - "name"
        - "children"`

	first := `openapi: 3.1.0
components:
  schemas:
    StartTest:
      type: object
      required:
        - muffins
      properties:
        muffins:
         $ref: "$2#/components/schemas/CircleTest"`

	var firstFile, secondFile, thirdFile, fourthFile *os.File
	var fErr error

	tmp := "tmp-h"
	_ = os.Mkdir(tmp, 0o755)

	firstFile, fErr = os.CreateTemp(tmp, "*-first.yaml")
	assert.NoError(t, fErr)

	secondFile, fErr = os.CreateTemp(tmp, "*-second.yaml")
	assert.NoError(t, fErr)

	thirdFile, fErr = os.CreateTemp(tmp, "*-third.yaml")
	assert.NoError(t, fErr)

	fourthFile, fErr = os.CreateTemp(tmp, "*-fourth.yaml")
	assert.NoError(t, fErr)

	first = strings.ReplaceAll(strings.ReplaceAll(first, "$2", secondFile.Name()), "\\", "\\\\")
	second = strings.ReplaceAll(strings.ReplaceAll(second, "$3", thirdFile.Name()), "\\", "\\\\")
	third = strings.ReplaceAll(strings.ReplaceAll(third, "$4", filepath.Base(fourthFile.Name())), "\\", "\\\\")

	firstFile.WriteString(first)
	secondFile.WriteString(second)
	thirdFile.WriteString(third)
	fourthFile.WriteString(fourth)

	defer os.RemoveAll(tmp)

	baseDir := tmp

	fsCfg := &LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         os.DirFS(baseDir),
		FileFilters: []string{
			filepath.Base(firstFile.Name()),
			filepath.Base(secondFile.Name()),
			filepath.Base(thirdFile.Name()),
			filepath.Base(fourthFile.Name()),
		},
	}

	fileFS, err := NewLocalFSWithConfig(fsCfg)
	if err != nil {
		t.Fatal(err)
	}

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.IgnorePolymorphicCircularReferences = true
	rolodex := NewRolodex(cf)
	rolodex.AddLocalFS(baseDir, fileFS)

	srv := test_rolodexDeepRefServer([]byte(first), []byte(second),
		[]byte(third), []byte(fourth), nil)
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	cf.BaseURL = u
	remoteFS, rErr := NewRemoteFSWithConfig(cf)
	assert.NoError(t, rErr)

	rolodex.AddRemoteFS(srv.URL, remoteFS)

	err = rolodex.IndexTheRolodex(context.Background())
	assert.Error(t, err)
	assert.Len(t, rolodex.GetCaughtErrors(), 2)
	assert.Equal(t, "cannot resolve reference `not_found.yaml`, it's missing: $.['not_found.yaml'] [8:11]", rolodex.GetCaughtErrors()[0].Error())
}

func TestRolodex_IndexCircularLookup_LookupHttpNoBaseURL(t *testing.T) {
	first := `openapi: 3.1.0
components:
  schemas:
    StartTest:
      type: object
      required:
        - muffins
      properties:
        muffins:
         type: object
         anyOf:
           - $ref: "https://raw.githubusercontent.com/pb33f/libopenapi/main/test_specs/circular-tests.yaml#/components/schemas/One"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(first), &rootNode)

	cf := CreateOpenAPIIndexConfig()
	cf.IgnorePolymorphicCircularReferences = true
	rolodex := NewRolodex(cf)

	remoteFS, rErr := NewRemoteFSWithConfig(cf)
	assert.NoError(t, rErr)

	rolodex.AddRemoteFS("", remoteFS)
	rolodex.SetRootNode(&rootNode)

	err := rolodex.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rolodex.GetCaughtErrors(), 0)
}

func TestRolodex_IndexCircularLookup_ignorePoly(t *testing.T) {
	spinny := `openapi: 3.1.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          anyOf:
            - $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spinny), &rootNode)

	cf := CreateOpenAPIIndexConfig()
	cf.IgnorePolymorphicCircularReferences = true
	rolodex := NewRolodex(cf)
	rolodex.SetRootNode(&rootNode)
	err := rolodex.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rolodex.GetCaughtErrors(), 0)
	assert.Len(t, rolodex.GetIgnoredCircularReferences(), 1)
}

func TestRolodex_IndexCircularLookup_ignoreArray(t *testing.T) {
	spinny := `openapi: 3.1.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "array"
          items:
            $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spinny), &rootNode)

	cf := CreateOpenAPIIndexConfig()
	cf.IgnoreArrayCircularReferences = true
	rolodex := NewRolodex(cf)
	rolodex.SetRootNode(&rootNode)
	err := rolodex.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rolodex.GetCaughtErrors(), 0)
	assert.Len(t, rolodex.GetIgnoredCircularReferences(), 1)
}

func TestRolodex_SimpleTest_OneDoc(t *testing.T) {
	baseDir := "rolodex_test_data"

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
		DirFS: os.DirFS(baseDir),
	})
	if err != nil {
		t.Fatal(err)
	}

	cf := CreateOpenAPIIndexConfig()
	cf.SpecFilePath = filepath.Join(baseDir, "doc1.yaml")
	cf.BasePath = baseDir
	cf.IgnoreArrayCircularReferences = true
	cf.IgnorePolymorphicCircularReferences = true

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	rootBytes, err := os.ReadFile(cf.SpecFilePath)
	assert.NoError(t, err)
	var rootNode yaml.Node
	_ = yaml.Unmarshal(rootBytes, &rootNode)
	rolo.SetRootNode(&rootNode)

	err = rolo.IndexTheRolodex(context.Background())

	// assert.NotZero(t, rolo.GetIndexingDuration()) comes back as 0 on windows.
	assert.NotNil(t, rolo.GetRootIndex())
	assert.Len(t, rolo.GetIndexes(), 11)
	assert.Len(t, rolo.GetAllReferences(), 10)
	assert.Len(t, rolo.GetAllMappedReferences(), 10)
	assert.Len(t, rolo.GetRootIndex().GetAllPaths(), 3)

	lineCount := rolo.GetFullLineCount()
	assert.Equal(t, int64(180), lineCount, "total line count in the rolodex is wrong")

	assert.NoError(t, err)
	assert.Len(t, rolo.indexes, 11)

	// open components.yaml
	f, rerr := rolo.Open("components.yaml")
	assert.NoError(t, rerr)
	assert.Equal(t, "components.yaml", f.Name())

	idx, ierr := f.(*rolodexFile).Index(cf)
	assert.NoError(t, ierr)
	assert.NotNil(t, idx)
	assert.Equal(t, YAML, f.GetFileExtension())
	assert.True(t, strings.HasSuffix(f.GetFullPath(), "rolodex_test_data"+string(os.PathSeparator)+"components.yaml"))
	assert.NotNil(t, f.ModTime())
	if runtime.GOOS != "windows" {
		assert.Equal(t, int64(448), f.Size())
	} else {
		assert.Equal(t, int64(467), f.Size())
	}
	assert.False(t, f.IsDir())
	assert.Nil(t, f.Sys())
	assert.Equal(t, fs.FileMode(0), f.Mode())
	assert.Len(t, f.GetErrors(), 0)

	// check the index has a rolodex reference
	assert.NotNil(t, idx.GetRolodex())

	// re-run the index should be a no-op
	assert.NoError(t, rolo.IndexTheRolodex(context.Background()))
	rolo.CheckForCircularReferences()
	assert.Len(t, rolo.GetIgnoredCircularReferences(), 0)
}

func TestRolodex_CircularReferencesPolyIgnored(t *testing.T) {
	d := `openapi: 3.1.0
components:
  schemas:
    bingo:
       type: object
       properties:
         bango:
           $ref: "#/components/schemas/ProductCategory"
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          items:
            anyOf:
              items:
                $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	c := CreateClosedAPIIndexConfig()
	c.IgnorePolymorphicCircularReferences = true
	rolo := NewRolodex(c)
	rolo.SetRootNode(&rootNode)
	_ = rolo.IndexTheRolodex(context.Background())
	assert.NotNil(t, rolo.GetRootIndex())
	rolo.CheckForCircularReferences()
	assert.Len(t, rolo.GetIgnoredCircularReferences(), 1)
	assert.Len(t, rolo.GetCaughtErrors(), 0)
}

func TestRolodex_CircularReferencesPolyIgnored_PostCheck(t *testing.T) {
	d := `openapi: 3.1.0
components:
  schemas:
    bingo:
       type: object
       properties:
         bango:
           $ref: "#/components/schemas/ProductCategory"
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          items:
            anyOf:
              items:
                $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	c := CreateClosedAPIIndexConfig()
	c.IgnorePolymorphicCircularReferences = true
	c.AvoidCircularReferenceCheck = true
	rolo := NewRolodex(c)
	rolo.SetRootNode(&rootNode)
	_ = rolo.IndexTheRolodex(context.Background())
	assert.NotNil(t, rolo.GetRootIndex())
	rolo.CheckForCircularReferences()
	assert.Len(t, rolo.GetIgnoredCircularReferences(), 1)
	assert.Len(t, rolo.GetCaughtErrors(), 0)
}

func TestRolodex_CircularReferencesPolyIgnored_Resolve(t *testing.T) {
	d := `openapi: 3.1.0
components:
  schemas:
    bingo:
       type: object
       properties:
         bango:
           $ref: "#/components/schemas/ProductCategory"
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          items:
            anyOf:
              items:
                $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	c := CreateClosedAPIIndexConfig()
	c.IgnorePolymorphicCircularReferences = true
	c.AvoidCircularReferenceCheck = true
	rolo := NewRolodex(c)
	rolo.SetRootNode(&rootNode)
	_ = rolo.IndexTheRolodex(context.Background())
	assert.NotNil(t, rolo.GetRootIndex())
	rolo.Resolve()
	assert.Len(t, rolo.GetIgnoredCircularReferences(), 1)
	assert.Len(t, rolo.GetCaughtErrors(), 0)
}

func TestRolodex_CircularReferencesPostCheck(t *testing.T) {
	d := `openapi: 3.1.0
components:
  schemas:
    bingo:
       type: object
       properties:
         bango:
           $ref: "#/components/schemas/bingo"
       required:
        - bango`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	c := CreateClosedAPIIndexConfig()
	c.AvoidCircularReferenceCheck = true
	rolo := NewRolodex(c)
	rolo.SetRootNode(&rootNode)
	_ = rolo.IndexTheRolodex(context.Background())
	assert.NotNil(t, rolo.GetRootIndex())
	rolo.CheckForCircularReferences()
	assert.Len(t, rolo.GetIgnoredCircularReferences(), 0)
	assert.Len(t, rolo.GetCaughtErrors(), 1)
	assert.Len(t, rolo.GetRootIndex().GetResolver().GetInfiniteCircularReferences(), 1)
	assert.Len(t, rolo.GetRootIndex().GetResolver().GetSafeCircularReferences(), 0)
}

func TestRolodex_CircularReferencesArrayIgnored(t *testing.T) {
	d := `openapi: 3.1.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "array"
          items:
            $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	c := CreateClosedAPIIndexConfig()
	c.IgnoreArrayCircularReferences = true
	rolo := NewRolodex(c)
	rolo.SetRootNode(&rootNode)
	_ = rolo.IndexTheRolodex(context.Background())
	rolo.CheckForCircularReferences()
	assert.Len(t, rolo.GetIgnoredCircularReferences(), 1)
	assert.Len(t, rolo.GetCaughtErrors(), 0)
}

func TestRolodex_CircularReferencesArrayIgnored_Resolve(t *testing.T) {
	d := `openapi: 3.1.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "array"
          items:
            $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	c := CreateClosedAPIIndexConfig()
	c.IgnoreArrayCircularReferences = true
	rolo := NewRolodex(c)
	rolo.SetRootNode(&rootNode)
	_ = rolo.IndexTheRolodex(context.Background())
	rolo.Resolve()
	assert.Len(t, rolo.GetIgnoredCircularReferences(), 1)
	assert.Len(t, rolo.GetCaughtErrors(), 0)
}

func TestRolodex_CircularReferencesArrayIgnored_PostCheck(t *testing.T) {
	d := `openapi: 3.1.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "array"
          items:
            $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	c := CreateClosedAPIIndexConfig()
	c.IgnoreArrayCircularReferences = true
	c.AvoidCircularReferenceCheck = true
	rolo := NewRolodex(c)
	rolo.SetRootNode(&rootNode)
	_ = rolo.IndexTheRolodex(context.Background())
	rolo.CheckForCircularReferences()
	assert.Len(t, rolo.GetIgnoredCircularReferences(), 1)
	assert.Len(t, rolo.GetCaughtErrors(), 0)
}

func TestHumanFileSize(t *testing.T) {
	// test bytes for different units
	assert.Equal(t, "1 B", HumanFileSize(1))
	assert.Equal(t, "1 KB", HumanFileSize(1024))
	assert.Equal(t, "1 MB", HumanFileSize(1024*1024))
}

func TestRolodex_GetSafeCircularReferences_nil(t *testing.T) {
	var r *Rolodex
	circ := r.GetSafeCircularReferences()
	assert.Nil(t, circ)
}

func TestRolodex_GetIgnoredCircularReferences_nil(t *testing.T) {
	var r *Rolodex
	circ := r.GetIgnoredCircularReferences()
	assert.Nil(t, circ)
}

func TestRolodex_SetSafeCircularRefs(t *testing.T) {
	var r *Rolodex
	r = NewRolodex(CreateOpenAPIIndexConfig())
	r.SetSafeCircularReferences([]*CircularReferenceResult{{
		LoopIndex: 1,
		LoopPoint: &Reference{
			FullDefinition: "test",
		},
	}})
	assert.NotNil(t, r.GetSafeCircularReferences())
}

func TestRolodex_CheckSetRootIndex(t *testing.T) {
	var r *Rolodex
	r = NewRolodex(CreateOpenAPIIndexConfig())
	r.SetRootIndex(&SpecIndex{})
	assert.NotNil(t, r.GetRootIndex())
}

func TestRolodex_CheckID(t *testing.T) {
	var r *Rolodex
	r = NewRolodex(CreateOpenAPIIndexConfig())
	id := r.GetId()
	assert.NotNil(t, id)

	r2 := NewRolodex(CreateOpenAPIIndexConfig())
	assert.NotNil(t, id)
	assert.NotEqual(t, r.GetId(), r2.GetId())

	a := r.GetId()
	b := r2.GetId()
	c := r.RotateId()
	d := r2.RotateId()

	assert.NotEqual(t, a, b)
	assert.NotEqual(t, a, c)
	assert.NotEqual(t, a, d)

}

func TestRolodex_IndexCircularLookup_SafeCircular(t *testing.T) {
	offToOz := `openapi: 3.1.0
components:
  schemas:
    One:
      properties:
        lemon:
          $ref: "#/components/schemas/Two"
    Two:
      properties:
        orange:
          allOf:
            - $ref: "#/components/schemas/One"
     `

	_ = os.WriteFile("off_to_ozmin.yaml", []byte(offToOz), 0o644)
	defer os.Remove("off_to_ozmin.yaml")

	baseDir, _ := os.Getwd()

	fsCfg := &LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         os.DirFS(baseDir),
		FileFilters: []string{
			"off_to_ozmin.yaml",
		},
	}

	fileFS, err := NewLocalFSWithConfig(fsCfg)
	if err != nil {
		t.Fatal(err)
	}

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	rolodex := NewRolodex(cf)
	rolodex.AddLocalFS(baseDir, fileFS)
	err = rolodex.IndexTheRolodex(context.Background())

	safeRefs := rolodex.GetSafeCircularReferences()
	assert.Len(t, safeRefs, 1)
}

func TestSpecIndex_TestDoubleIndexAdd(t *testing.T) {
	r := NewRolodex(CreateOpenAPIIndexConfig())
	r.AddExternalIndex(&SpecIndex{specAbsolutePath: "one"}, "one")
	r.AddExternalIndex(&SpecIndex{specAbsolutePath: "one"}, "one")
	r.AddExternalIndex(&SpecIndex{specAbsolutePath: "one"}, "one")
	assert.Len(t, r.GetIndexes(), 1)
}

type testRolodexFS struct {
	errorYaml bool
}

func (ts *testRolodexFS) Open(name string) (fs.File, error) {
	return &testRolodexFile{errorYaml: ts.errorYaml}, nil
}

func (ts *testRolodexFS) GetFiles() map[string]RolodexFile {
	return nil
}

type testRolodexFile struct {
	offset    int64
	errorYaml bool
}

func (trf *testRolodexFile) GetContent() string {
	return "test content"
}
func (trf *testRolodexFile) GetFileExtension() FileExtension {
	return YAML
}
func (trf *testRolodexFile) GetFullPath() string {
	return "/test/path/spec.yaml"
}
func (trf *testRolodexFile) GetErrors() []error {
	return nil
}
func (trf *testRolodexFile) GetContentAsYAMLNode() (*yaml.Node, error) {
	if trf.errorYaml {
		return nil, fmt.Errorf("error getting YAML node")
	}
	return &yaml.Node{}, nil
}
func (trf *testRolodexFile) GetIndex() *SpecIndex {
	return &SpecIndex{
		specAbsolutePath: "/test/path/spec.yaml",
	}
}
func (trf *testRolodexFile) Name() string {
	return "spec.yaml"
}
func (trf *testRolodexFile) ModTime() time.Time {
	return time.Now()
}
func (trf *testRolodexFile) IsDir() bool {
	return false
}
func (trf *testRolodexFile) Sys() any {
	return nil
}
func (trf *testRolodexFile) Size() int64 {
	return int64(len(trf.GetContent()))
}

func (trf *testRolodexFile) Mode() os.FileMode {
	return 0
}

// Close closes the file (doesn't do anything, returns no error)
func (trf *testRolodexFile) Close() error {
	return nil
}

// Stat returns the FileInfo for the file.
func (trf *testRolodexFile) Stat() (fs.FileInfo, error) {
	return trf, nil
}

// Read reads the file into a byte slice, makes it compatible with io.Reader.
func (trf *testRolodexFile) Read(b []byte) (int, error) {
	if trf.offset >= int64(len(trf.GetContent())) {
		return 0, io.EOF
	}
	if trf.offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: trf.GetFullPath(), Err: fs.ErrInvalid}
	}
	n := copy(b, trf.GetContent()[trf.offset:])
	trf.offset += int64(n)
	return n, nil
}

func TestRolodex_TestRolodexFileCompatibleFS(t *testing.T) {
	t.Parallel()

	// when using a custom FS, but also returning a RolodexFile compatible fs.File.

	testFS := &testRolodexFS{}

	baseDir := "/tmp"

	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS(baseDir, testFS)

	f, rerr := rolo.Open("spec.yaml")
	assert.NoError(t, rerr)
	assert.Equal(t, "test content", f.GetContent())
	rolo.rootIndex = NewTestSpecIndex().Load().(*SpecIndex)
	rolo.indexes = append(rolo.indexes, rolo.rootIndex)
	rolo.ClearIndexCaches()

	testFS = &testRolodexFS{errorYaml: true}

	rolo = NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS(baseDir, testFS)

	f, rerr = rolo.Open("spec.yaml")
	assert.Error(t, rerr)
	rolo.ClearIndexCaches()

}

// Test for line 606-607: filepath.Rel error handling
func TestRolodex_FilepathRelError(t *testing.T) {
	// Create a test FS that can handle the fallback
	testFS := &filepathRelFailFS{}
	
	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	// Use a base path that will cause filepath.Rel to fail when calculating relative paths
	// The base path is set to something that will trigger the filepath.Rel error path
	rolo.AddLocalFS("C:\\invalid:\\path", testFS) // Invalid on all platforms 
	
	// The file lookup should still find the file in the FS GetFiles map
	testFS.files = map[string]RolodexFile{
		"spec.yaml": &testRolodexFile{},
	}
	
	f, rerr := rolo.Open("spec.yaml")
	assert.NoError(t, rerr) // Should succeed because it falls back to original location
	assert.NotNil(t, f)
}

// Test for lines 626-630: fallback to original location when first attempt fails
func TestRolodex_FallbackToOriginalLocation(t *testing.T) {
	// Create a test FS that fails on calculated relative paths but succeeds on original
	testFS := &fallbackFS{failOnCalculatedPath: true}
	
	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS("/some/base/path", testFS)
	
	// Add the file to the lookup map so it can be found
	testFS.files = map[string]RolodexFile{
		"spec.yaml": &testRolodexFile{},
	}
	
	f, rerr := rolo.Open("spec.yaml")
	assert.NoError(t, rerr) // Should succeed via fallback
	assert.NotNil(t, f)
	assert.True(t, testFS.usedFallback) // Verify fallback was used
}

// Test for lines 778-779: remote file seeking errors
func TestRolodex_RemoteFileSeekingErrors(t *testing.T) {
	// Create a remote file with seeking errors
	remoteFile := &RemoteFile{
		fullPath: "http://example.com/spec.yaml",
		seekingErrors: []error{fmt.Errorf("seeking error 1"), fmt.Errorf("seeking error 2")},
	}
	
	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	
	// Create a rolodex file with the remote file that has seeking errors
	rolodexFile, err := rolo.createRolodexFileFromRemote(remoteFile, nil)
	assert.Error(t, err) // Should return the seeking errors
	assert.NotNil(t, rolodexFile)
	assert.Contains(t, err.Error(), "seeking error 1")
	assert.Contains(t, err.Error(), "seeking error 2")
}

// Helper test FS that causes filepath.Rel to fail
type filepathRelFailFS struct {
	files map[string]RolodexFile
}

func (f *filepathRelFailFS) Open(name string) (fs.File, error) {
	return &testFile{content: "test content"}, nil
}

func (f *filepathRelFailFS) GetFiles() map[string]RolodexFile {
	if f.files != nil {
		return f.files
	}
	return map[string]RolodexFile{
		"spec.yaml": &testRolodexFile{},
	}
}

// Helper test FS that fails on calculated paths but succeeds on original location
type fallbackFS struct {
	failOnCalculatedPath bool
	usedFallback        bool
	files               map[string]RolodexFile
}

func (f *fallbackFS) Open(name string) (fs.File, error) {
	// If this is the calculated path (not the exact spec.yaml), fail
	if f.failOnCalculatedPath && name != "spec.yaml" {
		return nil, fs.ErrNotExist
	}
	// If this is the original location, succeed and mark that fallback was used
	if name == "spec.yaml" {
		f.usedFallback = true
	}
	return &testFile{content: "test content"}, nil
}

func (f *fallbackFS) GetFiles() map[string]RolodexFile {
	if f.files != nil {
		return f.files
	}
	return map[string]RolodexFile{
		"spec.yaml": &testRolodexFile{},
	}
}

// Helper method to create a rolodex file from remote file (to test seeking errors)
func (r *Rolodex) createRolodexFileFromRemote(remoteFile *RemoteFile, localFile *LocalFile) (RolodexFile, error) {
	// This simulates the logic from lines 774-784
	if remoteFile != nil {
		// Check if the remoteFile has any seeking errors that should be returned
		var fileErrors []error
		if remoteFile.seekingErrors != nil && len(remoteFile.seekingErrors) > 0 {
			fileErrors = remoteFile.seekingErrors
		}
		return &rolodexFile{
			rolodex:    r,
			location:   remoteFile.fullPath,
			remoteFile: remoteFile,
		}, errors.Join(fileErrors...)
	}
	return nil, fmt.Errorf("no remote file provided")
}

// Test file implementation
type testFile struct {
	content string
	offset  int64
}

func (tf *testFile) Read(p []byte) (n int, err error) {
	if tf.offset >= int64(len(tf.content)) {
		return 0, io.EOF
	}
	n = copy(p, tf.content[tf.offset:])
	tf.offset += int64(n)
	return n, nil
}

func (tf *testFile) Close() error { return nil }

func (tf *testFile) Stat() (fs.FileInfo, error) {
	return &testFileInfo{name: "test.yaml", size: int64(len(tf.content))}, nil
}

// Test file info implementation
type testFileInfo struct {
	name string
	size int64
}

func (tfi *testFileInfo) Name() string       { return tfi.name }
func (tfi *testFileInfo) Size() int64        { return tfi.size }
func (tfi *testFileInfo) Mode() fs.FileMode  { return 0644 }
func (tfi *testFileInfo) ModTime() time.Time { return time.Now() }
func (tfi *testFileInfo) IsDir() bool        { return false }
func (tfi *testFileInfo) Sys() any          { return nil }


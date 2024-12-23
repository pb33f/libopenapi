// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var test_httpClient = &http.Client{Timeout: time.Duration(60) * time.Second}

func test_buildServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/file1.yaml") {
			rw.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
			_, _ = rw.Write([]byte(`"$ref": "./deeper/file2.yaml#/components/schemas/Pet"`))
			return
		}
		if req.URL.String() == "/deeper/file2.yaml" {
			rw.Header().Set("Last-Modified", "Wed, 21 Oct 2015 08:28:00 GMT")
			_, _ = rw.Write([]byte(`"$ref": "/deeper/even_deeper/file3.yaml#/components/schemas/Pet"`))
			return
		}

		if req.URL.String() == "/deeper/even_deeper/file3.yaml" {
			rw.Header().Set("Last-Modified", "Wed, 21 Oct 2015 10:28:00 GMT")
			_, _ = rw.Write([]byte(`"$ref": "../file2.yaml#/components/schemas/Pet"`))
			return
		}

		rw.Header().Set("Last-Modified", "Wed, 21 Oct 2015 12:28:00 GMT")

		if req.URL.String() == "/deeper/list.yaml" {
			_, _ = rw.Write([]byte(`"$ref": "../file2.yaml"`))
			return
		}

		if req.URL.String() == "/bag/list.yaml" {
			_, _ = rw.Write([]byte(`"$ref": "pocket/list.yaml"\n\n"$ref": "zip/things.yaml"`))
			return
		}

		if req.URL.String() == "/bag/pocket/list.yaml" {
			_, _ = rw.Write([]byte(`"$ref": "../list.yaml"\n\n"$ref": "../../file2.yaml"`))
			return
		}

		if req.URL.String() == "/bag/pocket/things.yaml" {
			_, _ = rw.Write([]byte(`"$ref": "list.yaml"`))
			return
		}

		if req.URL.String() == "/bag/zip/things.yaml" {
			_, _ = rw.Write([]byte(`"$ref": "list.yaml"`))
			return
		}

		if req.URL.String() == "/bag/zip/list.yaml" {
			_, _ = rw.Write([]byte(`"$ref": "../list.yaml"\n\n"$ref": "../../file1.yaml"\n\n"$ref": "more.yaml""`))
			return
		}

		if req.URL.String() == "/bag/zip/more.yaml" {
			_, _ = rw.Write([]byte(`"$ref": "../../deeper/list.yaml"\n\n"$ref": "../../bad.yaml"`))
			return
		}

		if req.URL.String() == "/bad.yaml" {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(`"error, cannot do the thing"`))
			return
		}

		_, _ = rw.Write([]byte(`OK`))
	}))
}

func TestNewRemoteFS_BasicCheck_Fail(t *testing.T) {
	server := test_buildServer()
	defer server.Close()

	// remoteFS := NewRemoteFS("https://raw.githubusercontent.com/digitalocean/openapi/main/specification/")
	remoteFS, _ := NewRemoteFSWithRootURL(server.URL)
	remoteFS.RemoteHandlerFunc = test_httpClient.Get

	file, err := remoteFS.Open("/file1.yaml")

	assert.Error(t, err)
	assert.Nil(t, file)
}

func TestNewRemoteFS_BasicCheck_Valid(t *testing.T) {
	server := test_buildServer()
	defer server.Close()

	// remoteFS := NewRemoteFS("https://raw.githubusercontent.com/digitalocean/openapi/main/specification/")
	remoteFS, _ := NewRemoteFSWithRootURL(server.URL)
	remoteFS.RemoteHandlerFunc = test_httpClient.Get

	file, err := remoteFS.Open("https://raw.githubusercontent.com/digitalocean/openapi/main/specification/file1.yaml")

	assert.NoError(t, err)

	bytes, rErr := io.ReadAll(file)
	assert.NoError(t, rErr)

	stat, _ := file.Stat()

	assert.Equal(t, "/digitalocean/openapi/main/specification/file1.yaml", stat.Name())
	assert.Equal(t, int64(53), stat.Size())
	assert.Len(t, bytes, 53)

	lastMod := stat.ModTime()
	assert.Equal(t, "2015-10-21 07:28:00 +0000 UTC", lastMod.UTC().String())
}

func TestNewRemoteFS_BasicCheck_NoScheme(t *testing.T) {
	server := test_buildServer()
	defer server.Close()

	remoteFS, _ := NewRemoteFSWithRootURL("")
	remoteFS.RemoteHandlerFunc = test_httpClient.Get

	file, err := remoteFS.Open("https://ding-dong-bing-bong.com/file1.yaml")

	assert.NoError(t, err)
	assert.Nil(t, file)
}

func TestNewRemoteFS_BasicCheck_Relative(t *testing.T) {
	server := test_buildServer()
	defer server.Close()

	remoteFS, _ := NewRemoteFSWithRootURL(server.URL)
	remoteFS.RemoteHandlerFunc = test_httpClient.Get

	file, err := remoteFS.Open("http://where-is-my-feet.com/deeper/file2.yaml")

	assert.NoError(t, err)

	bytes, rErr := io.ReadAll(file)
	assert.NoError(t, rErr)

	assert.Len(t, bytes, 64)

	stat, _ := file.Stat()

	assert.Equal(t, "/deeper/file2.yaml", stat.Name())
	assert.Equal(t, int64(64), stat.Size())

	lastMod := stat.ModTime()
	assert.Equal(t, "2015-10-21 08:28:00 +0000 UTC", lastMod.UTC().String())
}

func TestNewRemoteFS_BasicCheck_Relative_Deeper(t *testing.T) {
	server := test_buildServer()
	defer server.Close()

	cf := CreateOpenAPIIndexConfig()
	u, _ := url.Parse(server.URL)
	cf.BaseURL = u

	remoteFS, _ := NewRemoteFSWithConfig(cf)
	remoteFS.RemoteHandlerFunc = test_httpClient.Get

	file, err := remoteFS.Open("http://stop-being-a-dick-to-nature.com/deeper/even_deeper/file3.yaml")

	assert.NoError(t, err)

	bytes, rErr := io.ReadAll(file)
	assert.NoError(t, rErr)

	assert.Len(t, bytes, 47)

	stat, _ := file.Stat()

	assert.Equal(t, "/deeper/even_deeper/file3.yaml", stat.Name())
	assert.Equal(t, int64(47), stat.Size())
	assert.Equal(t, "/deeper/even_deeper/file3.yaml", file.(*RemoteFile).Name())
	assert.Equal(t, "file3.yaml", file.(*RemoteFile).GetFileName())
	assert.Len(t, file.(*RemoteFile).GetContent(), 47)
	assert.Equal(t, YAML, file.(*RemoteFile).GetFileExtension())
	assert.NotNil(t, file.(*RemoteFile).GetLastModified())
	assert.Len(t, file.(*RemoteFile).GetErrors(), 0)
	assert.Contains(t, file.(*RemoteFile).GetFullPath(), "/deeper/even_deeper/file3.yaml")
	assert.False(t, file.(*RemoteFile).IsDir())
	assert.Nil(t, file.(*RemoteFile).Sys())
	assert.Nil(t, file.(*RemoteFile).Close())

	lastMod := stat.ModTime()
	assert.Equal(t, "2015-10-21 10:28:00 +0000 UTC", lastMod.UTC().String())
}

func TestRemoteFile_NoContent(t *testing.T) {
	rf := &RemoteFile{}
	x, y := rf.GetContentAsYAMLNode()
	assert.Nil(t, x)
	assert.Error(t, y)
}

func TestRemoteFile_BadContent(t *testing.T) {
	rf := &RemoteFile{data: []byte("bad: data: on: a single: line: makes: for: unhappy: yaml"), index: &SpecIndex{}}
	x, y := rf.GetContentAsYAMLNode()
	assert.Nil(t, x)
	assert.Error(t, y)
}

func TestRemoteFile_GoodContent(t *testing.T) {
	rf := &RemoteFile{data: []byte("good: data"), index: &SpecIndex{}}
	x, y := rf.GetContentAsYAMLNode()
	assert.NotNil(t, x)
	assert.NoError(t, y)
	assert.NotNil(t, rf.index.root)

	// bad read
	rf.offset = -1
	d, err := io.ReadAll(rf)
	assert.Empty(t, d)
	assert.Error(t, err)
}

func TestRemoteFile_Index_AlreadySet(t *testing.T) {
	rf := &RemoteFile{data: []byte("good: data"), index: &SpecIndex{}}
	x, y := rf.Index(&SpecIndexConfig{})
	assert.NotNil(t, x)
	assert.NoError(t, y)
}

func TestRemoteFile_Index_BadContent_Recover(t *testing.T) {
	rf := &RemoteFile{data: []byte("no: sleep: until: the bugs: weep")}
	x, y := rf.Index(&SpecIndexConfig{})
	assert.NotNil(t, x)
	assert.NoError(t, y)
}

func TestRemoteFS_NoConfig(t *testing.T) {
	x, y := NewRemoteFSWithConfig(nil)
	assert.Nil(t, x)
	assert.Error(t, y)
}

func TestRemoteFS_SetRemoteHandler(t *testing.T) {
	h := func(url string) (*http.Response, error) {
		return nil, errors.New("nope")
	}
	cf := CreateClosedAPIIndexConfig()
	cf.RemoteURLHandler = h

	x, y := NewRemoteFSWithConfig(cf)
	assert.NotNil(t, x)
	assert.NoError(t, y)
	assert.NotNil(t, x.RemoteHandlerFunc)

	assert.NotNil(t, x.RemoteHandlerFunc)

	x.SetRemoteHandlerFunc(h)
	assert.NotNil(t, x.RemoteHandlerFunc)

	// run the handler
	i, n := x.RemoteHandlerFunc("http://www.google.com")
	assert.Nil(t, i)
	assert.Error(t, n)
	assert.Equal(t, "nope", n.Error())
}

func TestRemoteFS_NoConfigBadURL(t *testing.T) {
	x, y := NewRemoteFSWithRootURL("I am not a URL. I am a potato.: no.... // no.")
	assert.Nil(t, x)
	assert.Error(t, y)
}

func TestNewRemoteFS_Open_NoConfig(t *testing.T) {
	rfs := &RemoteFS{}
	x, y := rfs.Open("https://pb33f.io")
	assert.Nil(t, x)
	assert.Error(t, y)
}

func TestNewRemoteFS_Open_ConfigNotAllowed(t *testing.T) {
	rfs := &RemoteFS{indexConfig: CreateClosedAPIIndexConfig()}
	x, y := rfs.Open("https://pb33f.io")
	assert.Nil(t, x)
	assert.Error(t, y)
}

func TestNewRemoteFS_Open_BadURL(t *testing.T) {
	rfs := &RemoteFS{indexConfig: CreateOpenAPIIndexConfig()}
	x, y := rfs.Open("I am not a URL. I am a box of candy.. yum yum yum:: in my tum tum tum")
	assert.Nil(t, x)
	assert.Error(t, y)
}

func TestNewRemoteFS_RemoteBaseURL_RelativeRequest(t *testing.T) {
	cf := CreateOpenAPIIndexConfig()
	h := func(url string) (*http.Response, error) {
		return nil, fmt.Errorf("nope, not having it %s", url)
	}
	cf.RemoteURLHandler = h

	cf.BaseURL, _ = url.Parse("https://pb33f.io/the/love/machine")
	rfs, _ := NewRemoteFSWithConfig(cf)

	x, y := rfs.Open("https://pb33f.io/gib/gab/jib/jab.yaml")
	assert.Nil(t, x)
	assert.Error(t, y)
	assert.Equal(t, "nope, not having it https://pb33f.io/gib/gab/jib/jab.yaml", y.Error())
}

func TestNewRemoteFS_RemoteBaseURL_BadRequestButContainsBody(t *testing.T) {
	cf := CreateOpenAPIIndexConfig()
	h := func(url string) (*http.Response, error) {
		return &http.Response{}, fmt.Errorf("it's bad, but who cares %s", url)
	}
	cf.RemoteURLHandler = h

	cf.BaseURL, _ = url.Parse("https://pb33f.io/the/love/machine")
	rfs, _ := NewRemoteFSWithConfig(cf)

	x, y := rfs.Open("http://pb33f.io/woof.yaml")
	assert.Nil(t, x)
	assert.Error(t, y)
	assert.Equal(t, "it's bad, but who cares https://pb33f.io/woof.yaml", y.Error())
}

func TestNewRemoteFS_RemoteBaseURL_NoErrorNoResponse(t *testing.T) {
	cf := CreateOpenAPIIndexConfig()
	h := func(url string) (*http.Response, error) {
		return nil, nil // useless!
	}
	cf.RemoteURLHandler = h

	cf.BaseURL, _ = url.Parse("https://pb33f.io/the/love/machine")
	rfs, _ := NewRemoteFSWithConfig(cf)

	x, y := rfs.Open("https://pb33f.io/woof.yaml")
	assert.Nil(t, x)
	assert.Error(t, y)
	assert.Equal(t, "empty response from remote URL: https://pb33f.io/woof.yaml", y.Error())
}

func TestNewRemoteFS_RemoteBaseURL_200_NotOpenAPI(t *testing.T) {
	cf := CreateOpenAPIIndexConfig()
	h := func(url string) (*http.Response, error) {
		b := io.NopCloser(bytes.NewBuffer([]byte("not openapi")))
		return &http.Response{StatusCode: 200, Body: b}, nil
	}
	cf.RemoteURLHandler = h

	cf.BaseURL, _ = url.Parse("https://pb33f.io/the/love/machine")
	rfs, _ := NewRemoteFSWithConfig(cf)

	rolo := NewRolodex(cf)
	rolo.AddRemoteFS("https://pb33f.io/the/love/machine", rfs)
	f, e := rolo.Open("https://pb33f.io/woof.yaml")
	assert.NoError(t, e)
	c, err := f.(*rolodexFile).Index(cf)
	assert.Nil(t, c)
	assert.Error(t, err)

}

func TestNewRemoteFS_RemoteBaseURL_Error400(t *testing.T) {
	cf := CreateOpenAPIIndexConfig()
	h := func(url string) (*http.Response, error) {
		b := io.NopCloser(bytes.NewBuffer([]byte{}))
		return &http.Response{StatusCode: 400, Body: b}, nil
	}
	cf.RemoteURLHandler = h

	cf.BaseURL, _ = url.Parse("https://pb33f.io/the/love/machine")
	rfs, _ := NewRemoteFSWithConfig(cf)

	x, y := rfs.Open("https://pb33f.io/woof.yaml")
	assert.Nil(t, x)
	assert.Error(t, y)
	assert.Equal(t, "unable to fetch remote document 'https://pb33f.io/woof.yaml' (error 400)", y.Error())
}

func TestNewRemoteFS_RemoteBaseURL_ReadBodyFail(t *testing.T) {
	cf := CreateOpenAPIIndexConfig()
	h := func(url string) (*http.Response, error) {
		r := &http.Response{}
		r.Body = &LocalFile{offset: -1} // read will fail.
		return r, nil
	}
	cf.RemoteURLHandler = h

	cf.BaseURL, _ = url.Parse("https://pb33f.io/the/love/machine")
	rfs, _ := NewRemoteFSWithConfig(cf)

	x, y := rfs.Open("https://pb33f.io/woof.yaml")
	assert.Nil(t, x)
	assert.Error(t, y)
	assert.Equal(t, "error reading bytes from remote file 'https://pb33f.io/woof.yaml': "+
		"[read : invalid argument]", y.Error())
}

func TestNewRemoteFS_RemoteBaseURL_EmptySpecFailIndex(t *testing.T) {
	cf := CreateOpenAPIIndexConfig()
	h := func(url string) (*http.Response, error) {
		r := &http.Response{}
		r.Body = &LocalFile{data: []byte{}} // no bytes to read.
		return r, nil
	}
	cf.RemoteURLHandler = h

	cf.BaseURL, _ = url.Parse("https://pb33f.io/the/love/machine")
	rfs, _ := NewRemoteFSWithConfig(cf)

	x, y := rfs.Open("http://pb33f.io/woof.yaml")
	assert.NotNil(t, x)
	assert.Error(t, y)
	assert.Equal(t, "there is nothing in the spec, it's empty - so there is nothing to be done", y.Error())
}

func TestNewRemoteFS_Unsupported(t *testing.T) {
	cf := CreateOpenAPIIndexConfig()
	rfs, _ := NewRemoteFSWithConfig(cf)

	x, y := rfs.Open("http://pb33f.io/woof.png")
	assert.Nil(t, x)
	assert.Error(t, y)
	assert.Equal(t, "open http://pb33f.io/woof.png: invalid argument", y.Error())
}

func TestNewRemoteFS_BadURL(t *testing.T) {
	cf := CreateOpenAPIIndexConfig()
	rfs, _ := NewRemoteFSWithConfig(cf)

	x, y := rfs.Open("httpp://\r\nb33f.io/bingo.yaml")
	assert.Nil(t, x)
	assert.Error(t, y)
}

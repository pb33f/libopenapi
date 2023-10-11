// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rolodex

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var test_httpClient = &http.Client{Timeout: time.Duration(60) * time.Second}

func test_buildServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/file1.yaml" {
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

func TestNewRemoteFS_BasicCheck(t *testing.T) {

	server := test_buildServer()
	defer server.Close()

	//remoteFS := NewRemoteFS("https://raw.githubusercontent.com/digitalocean/openapi/main/specification/")
	remoteFS, _ := NewRemoteFS(server.URL)
	remoteFS.RemoteHandlerFunc = test_httpClient.Get

	file, err := remoteFS.Open("/file1.yaml")

	assert.NoError(t, err)

	bytes, rErr := io.ReadAll(file)
	assert.NoError(t, rErr)

	assert.Equal(t, "\"$ref\": \"\"./deeper/file2.yaml#/components/schemas/Pet\"", string(bytes))

	stat, _ := file.Stat()

	assert.Equal(t, "file1.yaml", stat.Name())
	assert.Equal(t, int64(54), stat.Size())

	lastMod := stat.ModTime()
	assert.Equal(t, "2015-10-21 07:28:00 +0000 GMT", lastMod.String())
}

func TestNewRemoteFS_BasicCheck_Relative(t *testing.T) {

	server := test_buildServer()
	defer server.Close()

	remoteFS, _ := NewRemoteFS(server.URL)
	remoteFS.RemoteHandlerFunc = test_httpClient.Get

	file, err := remoteFS.Open("/deeper/file2.yaml")

	assert.NoError(t, err)

	bytes, rErr := io.ReadAll(file)
	assert.NoError(t, rErr)

	assert.Equal(t, "\"$ref\": \"./deeper/even_deeper/file3.yaml#/components/schemas/Pet\"", string(bytes))

	stat, _ := file.Stat()

	assert.Equal(t, "/deeper/file2.yaml", stat.Name())
	assert.Equal(t, int64(65), stat.Size())

	lastMod := stat.ModTime()
	assert.Equal(t, "2015-10-21 08:28:00 +0000 GMT", lastMod.String())
}

func TestNewRemoteFS_BasicCheck_Relative_Deeper(t *testing.T) {

	server := test_buildServer()
	defer server.Close()

	remoteFS, _ := NewRemoteFS(server.URL)
	remoteFS.RemoteHandlerFunc = test_httpClient.Get

	file, err := remoteFS.Open("/deeper/even_deeper/file3.yaml")

	assert.NoError(t, err)

	bytes, rErr := io.ReadAll(file)
	assert.NoError(t, rErr)

	assert.Equal(t, "\"$ref\": \"../file2.yaml#/components/schemas/Pet\"", string(bytes))

	stat, _ := file.Stat()

	assert.Equal(t, "/deeper/even_deeper/file3.yaml", stat.Name())
	assert.Equal(t, int64(47), stat.Size())

	lastMod := stat.ModTime()
	assert.Equal(t, "2015-10-21 10:28:00 +0000 GMT", lastMod.String())
}

func TestNewRemoteFS_BasicCheck_SeekRelatives(t *testing.T) {

	server := test_buildServer()
	defer server.Close()

	remoteFS, _ := NewRemoteFS(server.URL)
	remoteFS.RemoteHandlerFunc = test_httpClient.Get

	file, err := remoteFS.Open("/bag/list.yaml")

	assert.Error(t, err)

	bytes, rErr := io.ReadAll(file)
	assert.NoError(t, rErr)

	assert.Equal(t, "\"$ref\": \"pocket/list.yaml\"\\n\\n\"$ref\": \"zip/things.yaml\"", string(bytes))

	stat, _ := file.Stat()

	assert.Equal(t, "/bag/list.yaml", stat.Name())
	assert.Equal(t, int64(55), stat.Size())

	lastMod := stat.ModTime()
	assert.Equal(t, "2015-10-21 12:28:00 +0000 GMT", lastMod.String())

	files := remoteFS.GetFiles()
	assert.Len(t, remoteFS.remoteErrors, 1)
	assert.Len(t, files, 10)

	// check correct files are in the cache
	assert.Equal(t, "/bag/list.yaml", files["/bag/list.yaml"].FullPath())
	assert.Equal(t, "list.yaml", files["/bag/list.yaml"].filename)

}

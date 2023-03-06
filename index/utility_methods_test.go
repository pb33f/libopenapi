// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
    "github.com/stretchr/testify/assert"
    "net/url"
    "testing"
)

func TestGenerateCleanSpecConfigBaseURL(t *testing.T) {
    u, _ := url.Parse("https://pb33f.io/things/stuff")
    path := "."
    assert.Equal(t, "https://pb33f.io/things/stuff",
        GenerateCleanSpecConfigBaseURL(u, path, false))
}

func TestGenerateCleanSpecConfigBaseURL_RelativeDeep(t *testing.T) {
    u, _ := url.Parse("https://pb33f.io/things/stuff/jazz/cakes/winter/oil")
    path := "../../../../foo/bar/baz/crap.yaml#thang"
    assert.Equal(t, "https://pb33f.io/things/stuff/foo/bar/baz",
        GenerateCleanSpecConfigBaseURL(u, path, false))

    assert.Equal(t, "https://pb33f.io/things/stuff/foo/bar/baz/crap.yaml#thang",
        GenerateCleanSpecConfigBaseURL(u, path, true))
}

func TestGenerateCleanSpecConfigBaseURL_NoBaseURL(t *testing.T) {

    u, _ := url.Parse("/things/stuff/jazz/cakes/winter/oil")
    path := "../../../../foo/bar/baz/crap.yaml#thang"
    assert.Equal(t, "/things/stuff/foo/bar/baz",
        GenerateCleanSpecConfigBaseURL(u, path, false))

    assert.Equal(t, "/things/stuff/foo/bar/baz/crap.yaml#thang",
        GenerateCleanSpecConfigBaseURL(u, path, true))
}

func TestGenerateCleanSpecConfigBaseURL_HttpStrip(t *testing.T) {

    u, _ := url.Parse(".")
    path := "http://thing.com/crap.yaml#thang"
    assert.Equal(t, "http://thing.com",
        GenerateCleanSpecConfigBaseURL(u, path, false))

    assert.Equal(t, "",
        GenerateCleanSpecConfigBaseURL(u, "crap.yaml#thing", true))
}

func TestSpecIndex_extractDefinitionRequiredRefProperties(t *testing.T) {
    c := CreateOpenAPIIndexConfig()
    idx := NewSpecIndexWithConfig(nil, c)
    assert.Nil(t, idx.extractDefinitionRequiredRefProperties(nil, nil))
}
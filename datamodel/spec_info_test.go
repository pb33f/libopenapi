// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
    "github.com/stretchr/testify/assert"
    "testing"
)

func TestSpecInfo_GetJSONParsingChannel(t *testing.T) {

    // dumb, but we need to ensure coverage is as high as we can make it.
    bchan := make(chan bool)
    si := &SpecInfo{JsonParsingChannel: bchan}
    assert.Equal(t, si.GetJSONParsingChannel(), bchan)

}

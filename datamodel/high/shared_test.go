// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package high

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "github.com/stretchr/testify/assert"
    "testing"
)

func TestExtractExtensions(t *testing.T) {
    n := make(map[low.KeyReference[string]]low.ValueReference[any])
    n[low.KeyReference[string]{
        Value: "pb33f",
    }] = low.ValueReference[any]{
        Value: "new cowboy in town",
    }
    ext := ExtractExtensions(n)
    assert.Equal(t, "new cowboy in town", ext["pb33f"])
}
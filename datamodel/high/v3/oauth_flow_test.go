// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pkg-base/libopenapi/orderedmap"
	"github.com/pkg-base/libopenapi/utils"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestOAuthFlow_MarshalYAML(t *testing.T) {
	scopes := orderedmap.New[string, string]()
	scopes.Set("chicken", "nuggets")
	scopes.Set("beefy", "soup")

	oflow := &OAuthFlow{
		AuthorizationUrl: "https://pb33f.io",
		TokenUrl:         "https://pb33f.io/token",
		RefreshUrl:       "https://pb33f.io/refresh",
		Scopes:           scopes,
	}

	rend, _ := oflow.Render()
	assert.NotNil(t, rend)

	desired := `authorizationUrl: https://pb33f.io
tokenUrl: https://pb33f.io/token
refreshUrl: https://pb33f.io/refresh
scopes:
    chicken: nuggets
    beefy: soup`

	// we can't check for equality, as the scopes map will be randomly ordered when created from scratch.
	assert.Len(t, desired, 149)

	// mutate
	oflow.Scopes = nil
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-burgers", utils.CreateStringNode("why not?"))
	oflow.Extensions = ext

	desired = `authorizationUrl: https://pb33f.io
tokenUrl: https://pb33f.io/token
refreshUrl: https://pb33f.io/refresh
x-burgers: why not?`

	rend, _ = oflow.Render()
	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

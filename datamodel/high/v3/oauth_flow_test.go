// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthFlow_MarshalYAML(t *testing.T) {
	t.Parallel()
	oflow := &OAuthFlow{
		AuthorizationUrl: "https://pb33f.io",
		TokenUrl:         "https://pb33f.io/token",
		RefreshUrl:       "https://pb33f.io/refresh",
		Scopes:           map[string]string{"chicken": "nuggets", "beefy": "soup"},
	}

	rend, err := oflow.Render()
	require.NoError(t, err)
	assert.NotEmpty(t, rend)

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
	oflow.Extensions = map[string]interface{}{"x-burgers": "why not?"}

	desired = `authorizationUrl: https://pb33f.io
tokenUrl: https://pb33f.io/token
refreshUrl: https://pb33f.io/refresh
x-burgers: why not?`

	rend, err = oflow.Render()
	require.NoError(t, err)
	assert.Equal(t, desired, strings.TrimSpace(string(rend)))

}

// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
    "github.com/stretchr/testify/assert"
    "strings"
    "testing"
)

func TestOAuthFlow_MarshalYAML(t *testing.T) {

    oflow := &OAuthFlow{
        AuthorizationUrl: "https://pb33f.io",
        TokenUrl:         "https://pb33f.io/token",
        RefreshUrl:       "https://pb33f.io/refresh",
        Scopes:           map[string]string{"chicken": "nuggets", "beefy": "soup"},
    }

    rend, _ := oflow.Render()

    desired := `authorizationUrl: https://pb33f.io
tokenUrl: https://pb33f.io/token
refreshUrl: https://pb33f.io/refresh
scopes:
    chicken: nuggets
    beefy: soup`

    assert.Equal(t, desired, strings.TrimSpace(string(rend)))

    // mutate
    oflow.Scopes = nil
    oflow.Extensions = map[string]interface{}{"x-burgers": "why not?"}

    desired = `authorizationUrl: https://pb33f.io
tokenUrl: https://pb33f.io/token
refreshUrl: https://pb33f.io/refresh
x-burgers: why not?`

    rend, _ = oflow.Render()
    assert.Equal(t, desired, strings.TrimSpace(string(rend)))

}

// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLink_MarshalYAML(t *testing.T) {
	t.Parallel()
	link := Link{
		OperationRef: "somewhere",
		OperationId:  "somewhereOutThere",
		Parameters: map[string]string{
			"over": "theRainbow",
		},
		RequestBody: "hello?",
		Description: "are you there?",
		Server: &Server{
			URL: "https://pb33f.io",
		},
	}

	dat, _ := link.Render()
	desired := `operationRef: somewhere
operationId: somewhereOutThere
parameters:
    over: theRainbow
requestBody: hello?
description: are you there?
server:
    url: https://pb33f.io`

	assert.Equal(t, desired, strings.TrimSpace(string(dat)))
}

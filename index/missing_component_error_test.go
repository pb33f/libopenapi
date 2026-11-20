// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

// A reference into a remote document that cannot be fetched must say why it is
// missing, rather than claim the component does not exist.
func TestMissingComponentError_ReportsFetchFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	remote := server.URL + "/api/openapi.yaml"
	spec := `openapi: 3.1.0
info: {title: Remote, version: '1'}
paths: {}
components:
  securitySchemes:
    external:
      $ref: '` + remote + `#/components/securitySchemes/ExternalSecurityThing'
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))

	config := CreateOpenAPIIndexConfig()
	config.AllowRemoteLookup = true
	remoteFS, err := NewRemoteFSWithConfig(config)
	require.NoError(t, err)
	rolodex := NewRolodex(config)
	rolodex.AddRemoteFS(server.URL, remoteFS)
	rolodex.SetRootNode(&root)
	indexErr := rolodex.IndexTheRolodex(t.Context())
	require.Error(t, indexErr)

	errs := rolodex.GetRootIndex().GetReferenceIndexErrors()
	require.Len(t, errs, 1)
	require.EqualError(t, errs[0], "component `#/components/securitySchemes/ExternalSecurityThing` cannot be resolved: "+
		"unable to fetch remote document '"+remote+"' (error 404 Not Found)")
	require.ErrorContains(t, indexErr, "(error 404 Not Found)")
}

// A local reference to a component that is absent keeps the original message.
func TestMissingComponentError_LocalComponent(t *testing.T) {
	spec := `openapi: 3.1.0
info: {title: Local, version: '1'}
paths: {}
components:
  schemas:
    Thing:
      $ref: '#/components/schemas/Missing'
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))
	idx := NewSpecIndexWithConfig(&root, CreateOpenAPIIndexConfig())
	errs := idx.GetReferenceIndexErrors()
	require.Len(t, errs, 1)
	require.EqualError(t, errs[0], "component `#/components/schemas/Missing` does not exist in the specification")
}

package index

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

const excessiveAliasDocument = `openapi: 3.1.1
info: {title: Alias limit, version: '1'}
paths: {}
components: {schemas: {Target: {type: string}}}
x-a: &a [value]
x-b: &b [*a, *a]
x-c: &c [*b, *b]
x-d: &d [*c, *c]
x-e: &e [*d, *d]
x-f: &f [*e, *e]
x-g: &g [*f, *f]
x-h: &h [*g, *g]
x-i: &i [*h, *h]
x-j: &j [*i, *i]
x-k: &k [*j, *j]
x-tail: {$ref: './unvisited.yaml'}
`

func TestExtractRefsAliasLimitSkipsRemainingSiblings(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(excessiveAliasDocument), &root))
	idx := NewSpecIndex(nil)
	require.Empty(t, idx.ExtractRefs(context.Background(), &root, nil, nil, 0, false, ""))
	require.Len(t, idx.GetReferenceIndexErrors(), 1)
	require.ErrorContains(t, idx.GetReferenceIndexErrors()[0], "document contains excessive aliasing")
	require.Empty(t, idx.GetRawReferencesSequenced(), "the tail reference must not be visited after the limit")
}

func TestRolodexReportsAliasLimitFromFileLoadedByRoot(t *testing.T) {
	for _, skipCircular := range []bool{false, true} {
		t.Run(fmt.Sprintf("skipCircular=%t", skipCircular), func(t *testing.T) {
			cfg := CreateOpenAPIIndexConfig()
			cfg.AvoidCircularReferenceCheck = skipCircular
			cfg.BaseURL, _ = url.Parse("https://example.com/")
			rolo := NewRolodex(cfg)
			remote, err := NewRemoteFSWithConfig(cfg)
			require.NoError(t, err)
			remote.RemoteHandlerFunc = func(location string) (*http.Response, error) {
				if location != "https://example.com/shared.yaml" {
					return nil, fmt.Errorf("unexpected remote lookup: %s", location)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(excessiveAliasDocument))}, nil
			}
			rolo.AddRemoteFS("https://example.com/", remote)
			require.Empty(t, remote.GetFiles(), "the external index must be loaded after the initial build queue")
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(`openapi: 3.1.1
info: {title: Root, version: '1'}
paths: {}
components:
  schemas:
    Target: {$ref: 'https://example.com/shared.yaml#/components/schemas/Target'}
`), &root))
			rolo.SetRootNode(&root)
			err = rolo.IndexTheRolodex(context.Background())
			require.ErrorContains(t, err, "document contains excessive aliasing")
			require.Len(t, rolo.GetIndexes(), 1)
			var resolvingErr *ResolvingError
			require.ErrorAs(t, err, &resolvingErr)
			require.NotNil(t, resolvingErr.Node)
			require.Contains(t, rolo.GetCaughtErrors(), resolvingErr)
		})
	}
}

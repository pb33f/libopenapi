package libopenapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

func TestBuildV3ModelReportsAliasLimit(t *testing.T) {
	// A small shared graph crosses the parser's alias ratio limit during indexing.
	// It has no references, so only the traversal limit can report this error.
	const source = `openapi: 3.1.1
info: {title: Aliases, version: '1'}
paths: {}
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
`
	for _, location := range []string{"root", "external"} {
		t.Run(location, func(t *testing.T) {
			for _, skipCircular := range []bool{false, true} {
				config := datamodel.NewDocumentConfiguration()
				config.SkipCircularReferenceCheck = skipCircular
				input := source
				if location == "external" {
					dir := t.TempDir()
					require.NoError(t, os.WriteFile(filepath.Join(dir, "shared.yaml"), []byte(source+"components: {schemas: {Target: {type: string}}}\n"), 0600))
					config.BasePath = dir
					config.AllowFileReferences = true
					input = "openapi: 3.1.1\ninfo: {title: Aliases, version: '1'}\npaths: {}\ncomponents: {schemas: {Target: {$ref: './shared.yaml#/components/schemas/Target'}}}\n"
				}
				doc, err := NewDocumentWithConfiguration([]byte(input), config)
				require.NoError(t, err)
				for range 2 {
					model, buildErr := doc.BuildV3Model()
					require.ErrorContains(t, buildErr, "document contains excessive aliasing")
					require.Nil(t, model)
				}
			}
		})
	}
}

func TestBuildV3ModelReportsMergeAliasLimit(t *testing.T) {
	const source = `openapi: 3.1.1
info: {title: Merge aliases, version: '1'}
paths: {}
x-a: &a {leaf: value}
x-b: &b {left: {<<: *a}, right: {<<: *a}}
x-c: &c {left: {<<: *b}, right: {<<: *b}}
x-d: &d {left: {<<: *c}, right: {<<: *c}}
x-e: &e {left: {<<: *d}, right: {<<: *d}}
x-f: &f {left: {<<: *e}, right: {<<: *e}}
x-g: &g {left: {<<: *f}, right: {<<: *f}}
x-h: &h {left: {<<: *g}, right: {<<: *g}}
x-i: &i {left: {<<: *h}, right: {<<: *h}}
x-j: &j {left: {<<: *i}, right: {<<: *i}}
x-k: &k {left: {<<: *j}, right: {<<: *j}}
`
	doc, err := NewDocument([]byte(source))
	require.NoError(t, err)
	for range 2 {
		model, buildErr := doc.BuildV3Model()
		require.ErrorContains(t, buildErr, "document contains excessive aliasing")
		require.Nil(t, model)
	}
}

package libopenapi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

func sharedResponseDocument() string {
	var source strings.Builder
	source.WriteString("openapi: 3.1.1\ninfo: {title: Shared responses, version: '1'}\nx-common:\n  errors: &errors\n")
	for i := range 20 {
		fmt.Fprintf(&source, "    '%d':\n      description: Error %d\n      content:\n        application/json:\n          schema:\n            type: object\n            properties:\n              code: {type: integer}\n              message: {type: string}\n", 400+i, i)
	}
	source.WriteString("paths:\n")
	for i := range 5000 {
		fmt.Fprintf(&source, "  /r%d:\n    get:\n      operationId: op%d\n      responses:\n        '200': {description: ok}\n        <<: *errors\n", i, i)
	}
	return source.String()
}

func TestBuildV3Model_AllowsTrustedSharedResponses(t *testing.T) {
	source := sharedResponseDocument()
	for _, external := range []bool{false, true} {
		for _, allow := range []bool{false, true} {
			t.Run(fmt.Sprintf("external=%t/allow=%t", external, allow), func(t *testing.T) {
				config := datamodel.NewDocumentConfiguration()
				input := source
				if external {
					config = externalRegressionConfig(t, map[string]string{"routes.yaml": source})
					input = "openapi: 3.1.1\ninfo: {title: Linked response, version: '1'}\npaths:\n  /linked: {$ref: './routes.yaml#/paths/~1r4999'}\n"
				}
				config.AllowExcessiveAliases = allow
				doc, err := NewDocumentWithConfiguration([]byte(input), config)
				require.NoError(t, err)
				model, err := doc.BuildV3Model()
				if !allow {
					require.ErrorContains(t, err, "AllowExcessiveAliases")
					require.Nil(t, model)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, model)
				require.True(t, model.Index.GetConfig().AllowExcessiveAliases)
				if external {
					require.Equal(t, 1, model.Model.Paths.PathItems.Len())
					require.Equal(t, "op4999", model.Model.Paths.PathItems.GetOrZero("/linked").Get.OperationId)
					require.Len(t, model.Index.GetRolodex().GetIndexes(), 1)
					require.Equal(t, 5000, model.Index.GetRolodex().GetIndexes()[0].GetOperationCount())
				} else {
					require.Equal(t, 5000, model.Model.Paths.PathItems.Len())
				}
				for item := range model.Model.Paths.PathItems.ValuesFromOldest() {
					require.Equal(t, 21, item.Get.Responses.Codes.Len())
					require.Equal(t, "Error 19", item.Get.Responses.Codes.GetOrZero("419").Description)
				}
			})
		}
	}
}

func TestBuildV2Model_PropagatesAliasPolicy(t *testing.T) {
	config := datamodel.NewDocumentConfiguration()
	config.AllowExcessiveAliases = true
	doc, err := NewDocumentWithConfiguration([]byte("swagger: '2.0'\ninfo: {title: Alias policy, version: '1'}\npaths: {}\n"), config)
	require.NoError(t, err)
	model, err := doc.BuildV2Model()
	require.NoError(t, err)
	require.True(t, model.Index.GetConfig().AllowExcessiveAliases)
	require.True(t, model.Index.GetConfig().ToDocumentConfiguration().AllowExcessiveAliases)
}

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

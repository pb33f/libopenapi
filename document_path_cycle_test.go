package libopenapi

import (
	"fmt"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

func TestBuildModel_PathReferenceCycleReturnsError(t *testing.T) {
	for _, version := range []string{"swagger: '2.0'", "openapi: 3.1.1"} {
		for _, skipCircular := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/skipCircular=%t", version, skipCircular), func(t *testing.T) {
				config := datamodel.NewDocumentConfiguration()
				config.SkipCircularReferenceCheck = skipCircular
				doc, err := NewDocumentWithConfiguration([]byte(version+`
info: {title: Path cycle, version: '1'}
x-a: {$ref: '#/x-b'}
x-b: {$ref: '#/x-a'}
paths:
  /a: {$ref: '#/x-a'}
`), config)
				require.NoError(t, err)
				for range 2 {
					if version == "swagger: '2.0'" {
						model, buildErr := doc.BuildV2Model()
						require.Nil(t, model)
						require.ErrorContains(t, buildErr, "circular reference chain")
					} else {
						model, buildErr := doc.BuildV3Model()
						require.Nil(t, model)
						require.ErrorContains(t, buildErr, "circular reference chain")
					}
				}
			})
		}
	}
}

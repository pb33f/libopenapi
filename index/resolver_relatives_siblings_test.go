package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

// A schema holding an external $ref next to sibling keys (allowed in OpenAPI 3.1) has its content
// replaced by the resolved target while extractRelativesWithState is still walking it. When the
// target has fewer entries than the referencing node, the walk must not index past the end.
func TestResolver_ExternalRefWithSiblingsLargerThanTarget(t *testing.T) {
	base := `openapi: 3.1.0
info: {title: base, version: '1'}
components:
  schemas:
    Base:
      type: object
      description: base
      required: [id]
      properties:
        id: {type: string}
`
	root := `openapi: 3.1.0
info: {title: root, version: '1'}
paths: {}
components:
  schemas:
    Thing:
      type: object
      description: thing
      required: [id, name]
      $ref: './base.yaml#/components/schemas/Base'
      properties:
        name: {type: string}
`
	tempDir := t.TempDir()
	rootPath := filepath.Join(tempDir, "root.yaml")
	require.NoError(t, os.WriteFile(rootPath, []byte(root), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "base.yaml"), []byte(base), 0o600))

	var rootNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(root), &rootNode))

	config := CreateOpenAPIIndexConfig()
	config.BasePath = tempDir
	config.SpecAbsolutePath = rootPath

	rolo := NewRolodex(config)
	rolo.SetRootNode(&rootNode)
	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{BaseDirectory: tempDir, IndexConfig: config})
	require.NoError(t, err)
	rolo.AddLocalFS(tempDir, fileFS)
	require.NoError(t, rolo.IndexTheRolodex(context.Background()))

	require.NotPanics(t, rolo.Resolve)
	require.Empty(t, rolo.GetRootIndex().GetResolver().GetResolvingErrors())
}

package index

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestLookupRolodexDoesNotWaitForTargetIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.yaml")
	cfg := CreateOpenAPIIndexConfig()
	cfg.BasePath = dir
	rolo := NewRolodex(cfg)
	cfg.Rolodex = rolo
	files, err := NewLocalFSWithConfig(&LocalFSConfig{BaseDirectory: dir, IndexConfig: cfg})
	assert.NoError(t, err)
	pending := &LocalFile{
		fullPath:         path,
		data:             []byte("components:\n  schemas:\n    Item:\n      type: object\n"),
		indexingComplete: make(chan struct{}),
	}
	files.Files.Store(path, pending)
	rolo.AddLocalFS(dir, files)
	defer close(pending.indexingComplete)
	var root yaml.Node
	assert.NoError(t, yaml.Unmarshal([]byte("components: {}"), &root))
	idx := NewSpecIndexWithConfig(&root, cfg)
	result := make(chan *Reference, 1)
	go func() {
		result <- idx.lookupRolodex(context.Background(), []string{path, "components/schemas/Item"})
	}()
	select {
	case ref := <-result:
		assert.NotNil(t, ref)
	case <-time.After(time.Second):
		t.Fatal("reference lookup waited for target indexing; mutually referencing files can deadlock")
	}
}

func TestLookupRolodexWholeFileUsesTargetIndex(t *testing.T) {
	for _, ready := range []bool{false, true} {
		name := "pending"
		if ready {
			name = "ready"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "target.yaml")
			cfg := CreateOpenAPIIndexConfig()
			cfg.BasePath = dir
			cfg.SpecAbsolutePath = filepath.Join(dir, "root.yaml")
			cfg.ResolveNestedRefsWithDocumentContext = true
			rolo := NewRolodex(cfg)
			cfg.Rolodex = rolo
			files, err := NewLocalFSWithConfig(&LocalFSConfig{BaseDirectory: dir, IndexConfig: cfg})
			require.NoError(t, err)
			target := &LocalFile{
				fullPath: path,
				data: []byte(`type: object
properties:
  item:
    $ref: '#/components/schemas/Item'
components:
  schemas:
    Item:
      type: string
`),
				indexingComplete: make(chan struct{}),
			}
			files.Files.Store(path, target)
			rolo.AddLocalFS(dir, files)
			defer close(target.indexingComplete)
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte("components:\n  schemas:\n    Item:\n      type: integer\n"), &root))
			idx := NewSpecIndexWithConfig(&root, cfg)
			targetNode, err := target.GetContentAsYAMLNode()
			require.NoError(t, err)
			targetCfg := *cfg
			targetCfg.SpecAbsolutePath = path
			targetIndex := NewSpecIndexWithConfig(targetNode, &targetCfg)
			if ready {
				target.index.Store(targetIndex)
				rolo.AddIndex(targetIndex)
			}
			result := make(chan *Reference, 1)
			go func() {
				result <- idx.lookupRolodex(context.Background(), []string{path})
			}()
			var ref *Reference
			select {
			case ref = <-result:
			case <-time.After(time.Second):
				t.Fatal("whole-file lookup waited for target indexing")
			}
			require.NotNil(t, ref)
			assert.Same(t, targetNode.Content[0], ref.Node)
			if !ready {
				assert.Nil(t, ref.Index, "a pending target must not inherit the caller's index")
				idx.cache.Store(path, ref)
				_, pendingIndex := idx.SearchIndexForReference(path)
				assert.Nil(t, pendingIndex)
				target.index.Store(targetIndex)
				rolo.AddIndex(targetIndex)
				_, cachedIndex := idx.SearchIndexForReference(path)
				assert.Same(t, targetIndex, cachedIndex, "a cached reference must use the target once its index is ready")
				return
			}
			assert.Same(t, targetIndex, ref.Index)
			resolver := NewResolver(idx)
			nested, nestedIndex, _ := resolver.searchReferenceWithContext(ref, &Reference{
				FullDefinition: "#/components/schemas/Item",
			})
			require.NotNil(t, nested)
			assert.Same(t, targetIndex, nestedIndex)
			var resolved struct {
				Type string `yaml:"type"`
			}
			require.NoError(t, nested.Node.Decode(&resolved))
			assert.Equal(t, "string", resolved.Type, "nested refs must resolve in the target document")
		})
	}
}

func TestExplicitWaitForIndexingStillWaitsForCompletion(t *testing.T) {
	for name, file := range map[string]func(chan struct{}) *rolodexFile{
		"local": func(done chan struct{}) *rolodexFile {
			return &rolodexFile{localFile: &LocalFile{indexingComplete: done}}
		},
		"remote": func(done chan struct{}) *rolodexFile {
			return &rolodexFile{remoteFile: &RemoteFile{indexingComplete: done}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			indexed := make(chan struct{})
			t.Cleanup(func() {
				select {
				case <-indexed:
				default:
					close(indexed)
				}
			})
			returned := make(chan struct{})
			go func() {
				file(indexed).WaitForIndexing()
				close(returned)
			}()
			select {
			case <-returned:
				t.Fatal("explicit wait returned before indexing completed")
			case <-time.After(10 * time.Millisecond):
			}
			close(indexed)
			select {
			case <-returned:
			case <-time.After(time.Second):
				t.Fatal("explicit wait did not return after indexing completed")
			}
			// Files without an indexing operation have nothing to wait for.
			file(nil).WaitForIndexing()
		})
	}
	(&rolodexFile{}).WaitForIndexing()
}

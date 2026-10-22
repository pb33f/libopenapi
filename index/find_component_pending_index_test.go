package index

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
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

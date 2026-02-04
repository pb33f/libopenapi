package index

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRelativeFilePath_PrefersBasePath(t *testing.T) {
	baseDir := t.TempDir()
	mapFS := fstest.MapFS{
		"resources/schemas/Item.yaml": {Data: []byte("type: object")},
	}

	localFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         mapFS,
	})
	require.NoError(t, err)

	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS(baseDir, localFS)

	cfg := CreateOpenAPIIndexConfig()
	cfg.BasePath = baseDir
	cfg.Rolodex = rolo
	idx := &SpecIndex{config: cfg, rolodex: rolo}

	defRoot := filepath.Join(baseDir, "resources", "paths")
	ref := "resources/schemas/Item.yaml"
	got := idx.ResolveRelativeFilePath(defRoot, ref)
	want := filepath.Join(baseDir, "resources", "schemas", "Item.yaml")
	assert.Equal(t, want, got)
}

func TestResolveRelativeFilePath_Fallback_NoIndex(t *testing.T) {
	var idx *SpecIndex
	got := idx.ResolveRelativeFilePath("/base", "file.yaml")
	want, _ := filepath.Abs(utils.CheckPathOverlap("/base", "file.yaml", string(os.PathSeparator)))
	assert.Equal(t, want, got)
}

func TestResolveRelativeFilePath_Fallback_BaseURL(t *testing.T) {
	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	u := &url.URL{Scheme: "https", Host: "example.com"}
	cfg := CreateOpenAPIIndexConfig()
	cfg.BaseURL = u
	cfg.Rolodex = rolo
	idx := &SpecIndex{config: cfg, rolodex: rolo}
	got := idx.ResolveRelativeFilePath("/base", "file.yaml")
	want, _ := filepath.Abs(utils.CheckPathOverlap("/base", "file.yaml", string(os.PathSeparator)))
	assert.Equal(t, want, got)
}

func TestPathExistsInFS_LocalFS_DirFS(t *testing.T) {
	baseDir := t.TempDir()
	mapFS := fstest.MapFS{
		"resources/x.yaml": {Data: []byte("test")},
	}
	localFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         mapFS,
	})
	require.NoError(t, err)

	absPath := filepath.Join(baseDir, "resources", "x.yaml")
	assert.True(t, pathExistsInFS(baseDir, localFS, absPath))

	absPath = filepath.Join(filepath.Dir(baseDir), "resources", "x.yaml")
	assert.False(t, pathExistsInFS(baseDir, localFS, absPath))
}

func TestPathExistsInFS_LocalFS_OS(t *testing.T) {
	baseDir := t.TempDir()
	absPath := filepath.Join(baseDir, "file.yaml")
	require.NoError(t, os.WriteFile(absPath, []byte("test"), 0o600))

	localFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
	})
	require.NoError(t, err)

	assert.True(t, pathExistsInFS(baseDir, localFS, absPath))
}

func TestPathExistsInFS_NonLocalFS(t *testing.T) {
	baseDir := "/base"
	fsys := fstest.MapFS{
		"resources/x.yaml": {Data: []byte("test")},
	}
	absPath := filepath.Join(baseDir, "resources", "x.yaml")
	assert.True(t, pathExistsInFS(baseDir, fsys, absPath))
}

func TestResolverResolveLocalRefPath(t *testing.T) {
	baseDir := t.TempDir()
	mapFS := fstest.MapFS{
		"resources/x.yaml": {Data: []byte("test")},
	}
	localFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         mapFS,
	})
	require.NoError(t, err)

	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS(baseDir, localFS)

	cfg := CreateOpenAPIIndexConfig()
	cfg.BasePath = baseDir
	cfg.Rolodex = rolo
	idx := &SpecIndex{config: cfg, rolodex: rolo}
	resolver := NewResolver(idx)

	got := resolver.resolveLocalRefPath(filepath.Join(baseDir, "resources"), "resources/x.yaml")
	want := idx.ResolveRelativeFilePath(filepath.Join(baseDir, "resources"), "resources/x.yaml")
	assert.Equal(t, want, got)

	var nilResolver *Resolver
	gotFallback := nilResolver.resolveLocalRefPath("/base", "file.yaml")
	wantFallback, _ := filepath.Abs(utils.CheckPathOverlap("/base", "file.yaml", string(os.PathSeparator)))
	assert.Equal(t, wantFallback, gotFallback)
}

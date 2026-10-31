package index

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

type sharedIndexedFS struct {
	files map[string]RolodexFile
	lazy  bool
}

func TestRolodexMissingLocalFile(t *testing.T) {
	// LocalFS can return a typed nil for an unavailable discriminator target.
	var missing *LocalFile
	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	file, errs := rolo.asLocalFile(missing, "missing.yaml")
	require.Nil(t, file)
	require.Empty(t, errs)
}

type controlledIndexedFile struct {
	*LocalFile
	result *SpecIndex
	err    error
	calls  int
}

func (f *controlledIndexedFile) Index(*SpecIndexConfig) (*SpecIndex, error) {
	f.calls++
	if f.result != nil {
		f.index.Store(f.result)
	}
	return f.result, f.err
}

func standaloneIndexedFile(t *testing.T, content string) *LocalFile {
	t.Helper()
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("type: object\n"), &root))
	file := &LocalFile{data: []byte(content), name: "schema.yaml", filename: "schema.yaml", fullPath: filepath.Join(t.TempDir(), "schema.yaml"), extension: YAML}
	file.index.Store(NewSpecIndex(&root))
	return file
}

func TestRolodexOwnsStandaloneAndLateIndexes(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(map[bool]string{false: "cached", true: "returned-by-index"}[late], func(t *testing.T) {
			original := standaloneIndexedFile(t, "type: object\nproperties:\n  id: {type: string}\n")
			file := &controlledIndexedFile{LocalFile: original, result: original.GetIndex()}
			if late {
				file.LocalFile = &LocalFile{data: original.data, name: original.name, filename: original.filename, fullPath: original.fullPath, extension: YAML}
			}
			files := &sharedIndexedFS{files: map[string]RolodexFile{"schema.yaml": file}}
			cfg := CreateOpenAPIIndexConfig()
			cfg.BasePath = filepath.Dir(original.fullPath)
			rolo := NewRolodex(cfg)
			rolo.AddLocalFS(cfg.BasePath, files)
			require.NoError(t, rolo.IndexTheRolodex(context.Background()))
			owned, err := rolo.Open(original.fullPath)
			require.NoError(t, err)
			require.Same(t, rolo, owned.GetIndex().GetRolodex())
			require.NotSame(t, original.GetIndex(), owned.GetIndex())
			if late {
				require.Equal(t, 1, file.calls)
			} else {
				require.Zero(t, file.calls, "do not ask a cached foreign file to index under another owner")
			}
		})
	}
}

func TestRolodexReportsIndexAndOwnedParseErrors(t *testing.T) {
	for _, failure := range []string{"index-error", "nil-index", "empty-content"} {
		t.Run(failure, func(t *testing.T) {
			file := &controlledIndexedFile{LocalFile: &LocalFile{name: "schema.yaml", fullPath: filepath.Join(t.TempDir(), "schema.yaml"), data: []byte("type: string\n")}}
			switch failure {
			case "index-error":
				file.err = errors.New("custom index failed")
			case "empty-content":
				file.LocalFile = standaloneIndexedFile(t, "")
			}
			cfg := CreateOpenAPIIndexConfig()
			cfg.BasePath = filepath.Dir(file.fullPath)
			rolo := NewRolodex(cfg)
			rolo.AddLocalFS(cfg.BasePath, &sharedIndexedFS{files: map[string]RolodexFile{"schema.yaml": file}})
			require.Error(t, rolo.IndexTheRolodex(context.Background()))
			if failure == "empty-content" {
				opened, err := rolo.Open(file.fullPath)
				require.Error(t, err)
				require.Nil(t, opened)
			}
		})
	}
}

type synchronizedContentFile struct {
	*LocalFile
	reads atomic.Int32
}

func (f *synchronizedContentFile) GetContent() string {
	f.reads.Add(1)
	return f.LocalFile.GetContent()
}

func TestRolodexConcurrentForeignFileCopiesShareOneIndex(t *testing.T) {
	const callers = 8
	var workers sync.WaitGroup
	source := &synchronizedContentFile{LocalFile: standaloneIndexedFile(t, "type: object\nproperties:\n  id: {type: string}\n")}
	cfg := CreateOpenAPIIndexConfig()
	cfg.IgnoreArrayCircularReferences = true
	cfg.IgnorePolymorphicCircularReferences = true
	rolo := NewRolodex(cfg)
	results := make([]*LocalFile, callers)
	errs := make([]error, callers)
	start := make(chan struct{})
	workers.Add(callers)
	for i := range callers {
		go func() { defer workers.Done(); <-start; results[i], errs[i] = rolo.ownIndexedFile(source) }()
	}
	close(start)
	workers.Wait()
	require.Equal(t, int32(1), source.reads.Load(), "concurrent opens must parse the source only once")
	for i := range callers {
		require.NoError(t, errs[i])
		results[i].WaitForIndexing()
		require.Same(t, results[0], results[i])
	}
	require.Len(t, rolo.GetIndexes(), 1)
	idx := results[0].GetIndex()
	require.Same(t, rolo, idx.GetRolodex())
	require.True(t, idx.GetResolver().IgnoreArray)
	require.True(t, idx.GetResolver().IgnorePoly)
	again, err := rolo.ownIndexedFile(source)
	require.NoError(t, err)
	require.Same(t, results[0], again)
}

func TestRolodexOwnsCachedRemoteFile(t *testing.T) {
	source := standaloneIndexedFile(t, "type: string\n")
	remote := &RemoteFile{data: source.data, filename: "schema.yaml", name: "schema.yaml", fullPath: "https://example.com/schema.yaml", extension: YAML}
	remote.index.Store(source.GetIndex())
	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	owned, err := rolo.wrapRemoteRolodexFile(remote)
	require.NoError(t, err)
	require.Same(t, rolo, owned.GetIndex().GetRolodex())
	require.NotSame(t, remote.GetIndex().GetRootNode(), owned.GetIndex().GetRootNode())
	require.Equal(t, remote.GetFullPath(), owned.GetFullPath())
}

func (s *sharedIndexedFS) Open(name string) (fs.File, error) {
	if file := s.files[filepath.Base(name)]; file != nil {
		return file.(fs.File), nil
	}
	return nil, fs.ErrNotExist
}

func (s *sharedIndexedFS) GetFiles() map[string]RolodexFile {
	if s.lazy {
		return nil
	}
	return s.files
}

// customIndexedFile preserves the same cached index as LocalFile through a
// custom RolodexFile implementation, as Vacuum's lazy filesystem does.
type customIndexedFile struct{ *LocalFile }

type blockedContentFile struct {
	RolodexFile
	fs.File
	entered chan struct{}
	resume  chan struct{}
}

func (f *blockedContentFile) GetContent() string {
	close(f.entered)
	<-f.resume
	return f.RolodexFile.GetContent()
}

func TestRolodexPendingOwnedFileRetainsNodeIdentity(t *testing.T) {
	files, dir, root := sharedCircularFiles(t, true)
	indexSharedFiles(t, files, dir, root)
	blocked := &blockedContentFile{RolodexFile: files.files["b.yaml"], File: files.files["b.yaml"].(fs.File), entered: make(chan struct{}), resume: make(chan struct{})}
	files.files["b.yaml"] = blocked
	files.lazy = true
	cfg := CreateOpenAPIIndexConfig()
	cfg.BasePath = dir
	cfg.SpecAbsolutePath = filepath.Join(dir, "root.yaml")
	rolo := NewRolodex(cfg)
	rolo.AddLocalFS(dir, files)
	var callerRoot yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("components: {}\n"), &callerRoot))
	caller := NewSpecIndexWithConfig(&callerRoot, cfg)
	finished := make(chan error, 1)
	go func() { _, err := rolo.Open(filepath.Join(dir, "a.yaml")); finished <- err }()
	resume := sync.OnceFunc(func() { close(blocked.resume) })
	defer resume()
	select {
	case <-blocked.entered:
	case <-time.After(time.Second):
		t.Fatal("foreign indexing did not reach the second file")
	}
	// The recursive lookup must expose the owned parsed tree without waiting
	// for the A -> B -> A index construction to finish.
	ref := caller.lookupRolodex(context.Background(), []string{filepath.Join(dir, "a.yaml"), "A"})
	require.NotNil(t, ref)
	require.Nil(t, ref.Index)
	resume()
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("cross-file ownership indexing deadlocked")
	}
	file, err := rolo.Open(filepath.Join(dir, "a.yaml"))
	require.NoError(t, err)
	ready := file.GetIndex().FindComponent(context.Background(), "#/A")
	require.NotNil(t, ready)
	require.Same(t, ref.Node, ready.Node, "pending and completed lookups must use the same YAML nodes")
}

func sharedCircularFiles(t *testing.T, custom bool) (*sharedIndexedFS, string, []byte) {
	t.Helper()
	dir, err := filepath.Abs("../test_specs/circular_cross_file/orig")
	require.NoError(t, err)
	files := &sharedIndexedFS{files: make(map[string]RolodexFile)}
	for _, name := range []string{"a.yaml", "b.yaml"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		file := &LocalFile{data: data, name: name, filename: name, fullPath: filepath.Join(dir, name), extension: YAML}
		if custom {
			files.files[name] = &customIndexedFile{file}
		} else {
			files.files[name] = file
		}
	}
	root, err := os.ReadFile(filepath.Join(dir, "root.yaml"))
	require.NoError(t, err)
	return files, dir, root
}

func indexSharedFiles(t *testing.T, files *sharedIndexedFS, dir string, root []byte) *Rolodex {
	t.Helper()
	cfg := CreateOpenAPIIndexConfig()
	cfg.BasePath = dir
	cfg.SpecFilePath = "root.yaml"
	r := NewRolodex(cfg)
	r.AddLocalFS(dir, files)
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal(root, &node))
	r.SetRootNode(&node)
	require.NoError(t, r.IndexTheRolodex(context.Background()))
	return r
}

func TestRolodexSharedIndexedFilesKeepDocumentOwnership(t *testing.T) {
	for _, custom := range []bool{false, true} {
		for _, lazy := range []bool{false, true} {
			name := "local"
			if custom {
				name = "custom"
			}
			if lazy {
				name += "-lazy"
			}
			t.Run(name, func(t *testing.T) {
				files, dir, root := sharedCircularFiles(t, custom)
				first := indexSharedFiles(t, files, dir, root)
				firstA := files.files["a.yaml"].GetIndex()
				firstB := files.files["b.yaml"].GetIndex()
				firstResolver := firstA.GetResolver()
				files.lazy = lazy
				second := indexSharedFiles(t, files, dir, root)
				require.Same(t, firstA, files.files["a.yaml"].GetIndex())
				require.Same(t, firstResolver, firstA.GetResolver(), "building another document must not reset the original resolver")
				for _, name := range []string{"a.yaml", "b.yaml"} {
					file, err := second.Open(filepath.Join(dir, name))
					require.NoError(t, err)
					file.WaitForIndexing()
					idx := file.GetIndex()
					require.NotSame(t, files.files[name].GetIndex(), idx)
					require.Same(t, second, idx.GetRolodex())
					node, err := file.GetContentAsYAMLNode()
					require.NoError(t, err)
					require.Same(t, idx.GetRootNode(), node)
					require.NotSame(t, files.files[name].GetIndex().GetRootNode(), node)
				}
				require.Len(t, second.GetIndexes(), 2, "lazy indexes must survive batch registration")
				var wg sync.WaitGroup
				wg.Add(1)
				go func() { defer wg.Done(); first.Resolve() }()
				for range 100 {
					second.GetRootIndex().GetCircularReferences()
					second.GetSafeCircularReferences()
					second.GetIgnoredCircularReferences()
				}
				wg.Wait()
				require.Same(t, first, firstB.GetRolodex())
				second.Release()
				require.Nil(t, second.ownedFiles)
			})
		}
	}
}

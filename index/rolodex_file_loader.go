// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"context"
	"github.com/pb33f/libopenapi/datamodel"
	"gopkg.in/yaml.v3"
)

// LocalFS is a file system that indexes local files.
type LocalFS struct {
	fsConfig            *LocalFSConfig
	indexConfig         *SpecIndexConfig
	entryPointDirectory string
	baseDirectory       string
	Files               sync.Map
	extractedFiles      map[string]RolodexFile
	logger              *slog.Logger
	readingErrors       []error
	rolodex             *Rolodex
	processingFiles     sync.Map
}

// GetFiles returns the files that have been indexed. A map of RolodexFile objects keyed by the full path of the file.
func (l *LocalFS) GetFiles() map[string]RolodexFile {
	files := make(map[string]RolodexFile)
	l.Files.Range(func(key, value interface{}) bool {
		files[key.(string)] = value.(*LocalFile)
		return true
	})
	l.extractedFiles = files
	return files
}

// GetErrors returns any errors that occurred during the indexing process.
func (l *LocalFS) GetErrors() []error {
	return l.readingErrors
}

type waiterLocal struct {
	f         string
	done      bool
	file      *LocalFile
	listeners int
	error     error
	mu        sync.RWMutex
	cond      *sync.Cond
}

// In LocalFS, add a map[uint64]map[string]struct{} to track goroutine-file
var processingMap sync.Map // goroutine id => set of file names

func (l *LocalFS) OpenWithContext(ctx context.Context, name string) (fs.File, error) {
	if l.indexConfig != nil && !l.indexConfig.AllowFileLookup {
		return nil, &fs.PathError{
			Op: "open", Path: name,
			Err: fmt.Errorf("file lookup for '%s' not allowed, set the index configuration "+
				"to AllowFileLookup to be true", name),
		}
	}

	if !filepath.IsAbs(name) {
		name, _ = filepath.Abs(filepath.Join(l.baseDirectory, name))
	}

	if f, ok := l.Files.Load(name); ok {
		return f.(*LocalFile), nil
	}

	// Only enter new-file logic if DirFS is not set
	if l.fsConfig != nil && l.fsConfig.DirFS == nil {

		// If we're processing, we need to block and wait for the file to be processed
		if r, ko := l.processingFiles.Load(name); ko {
			wait := r.(*waiterLocal)

			wait.mu.Lock()
			wait.listeners++
			l.logger.Debug("[rolodex file loader]: waiting for existing OS load to complete", "file", name, "listeners", wait.listeners)
			for !wait.done {
				wait.cond.Wait()
			}
			f := wait.file
			e := wait.error
			wait.listeners--
			l.logger.Debug("[rolodex file loader]: waiting done, OS load completed, returning file", "file", name, "listeners", wait.listeners)
			wait.mu.Unlock()
			return f, e
		}

		processingWaiter := &waiterLocal{f: name}
		processingWaiter.cond = sync.NewCond(&processingWaiter.mu)

		processingWaiter.mu.Lock()

		l.processingFiles.Store(name, processingWaiter)

		var extractedFile *LocalFile
		var extErr error
		l.logger.Debug("[rolodex file loader]: extracting file from OS", "file", name)
		extractedFile, extErr = l.extractFile(name)

		if extErr != nil {
			processingWaiter.error = extErr
			processingWaiter.done = true
			if processingWaiter.cond != nil {
				processingWaiter.cond.Broadcast()
			}
			l.processingFiles.Delete(name)
			processingWaiter.mu.Unlock()

			return nil, extErr
		}
		if extractedFile != nil && l.indexConfig != nil {
			copiedCfg := *l.indexConfig
			copiedCfg.SpecAbsolutePath = name
			copiedCfg.AvoidBuildIndex = true
			copiedCfg.SpecInfo = nil

			idx, _ := extractedFile.IndexWithContext(ctx, &copiedCfg)
			if idx != nil && l.rolodex != nil {
				idx.rolodex = l.rolodex
			}

			if idx != nil {
				resolver := NewResolver(idx)
				idx.resolver = resolver
				idx.BuildIndex()
			}
			if len(extractedFile.data) > 0 {
				l.logger.Debug("[rolodex file loader]: successfully loaded and indexed file", "file", name)
			}
			if l.rolodex != nil {
				l.rolodex.AddIndex(idx)
			}
		}

		processingWaiter.file = extractedFile
		processingWaiter.error = extErr
		processingWaiter.done = true
		if processingWaiter.cond != nil {
			processingWaiter.cond.Broadcast()
		}
		l.processingFiles.Delete(name)
		processingWaiter.mu.Unlock()

		return extractedFile, nil
	}

	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// Open opens a file, returning it or an error. If the file is not found, the error is of type *PathError.
func (l *LocalFS) Open(name string) (fs.File, error) {
	return l.OpenWithContext(context.Background(), name)
}

// LocalFile is a file that has been indexed by the LocalFS. It implements the RolodexFile interface.
type LocalFile struct {
	filename      string
	name          string
	extension     FileExtension
	data          []byte
	fullPath      string
	lastModified  time.Time
	readingErrors []error
	index         atomic.Value
	parsed        *yaml.Node
	offset        int64
	parseMutex    sync.Mutex
	indexOnce     sync.Once
}

// GetIndex returns the *SpecIndex for the file.
func (l *LocalFile) GetIndex() *SpecIndex {
	if v := l.index.Load(); v != nil {
		return v.(*SpecIndex)
	}
	return nil
}

// Index returns the *SpecIndex for the file. If the index has not been created, it will be created (indexed)
func (l *LocalFile) Index(config *SpecIndexConfig) (*SpecIndex, error) {
	return l.IndexWithContext(context.Background(), config)
}

// // IndexWithContext returns the *SpecIndex for the file. If the index has not been created, it will be created (indexed), also supplied context
func (l *LocalFile) IndexWithContext(ctx context.Context, config *SpecIndexConfig) (*SpecIndex, error) {
	var result *SpecIndex
	var resultErr error
	l.indexOnce.Do(func() {
		if idx := l.GetIndex(); idx != nil {
			result = idx
			return
		}

		content := l.data
		// first, we must parse the content of the file,
		// the check is bypassed, so as long as it's readable, we're good.
		info, _ := datamodel.ExtractSpecInfoWithDocumentCheck(content, true)
		if config.SpecInfo == nil {
			config.SpecInfo = info
		}
		index := NewSpecIndexWithConfigAndContext(ctx, info.RootNode, config)
		index.specAbsolutePath = l.fullPath
		l.index.Store(index)
		result = index
	})
	if v := l.index.Load(); v != nil {
		result = v.(*SpecIndex)
	}
	return result, resultErr

}

// GetContent returns the content of the file as a string.
func (l *LocalFile) GetContent() string {
	return string(l.data)
}

// GetContentAsYAMLNode returns the content of the file as a *yaml.Node. If something went wrong
// then an error is returned.
func (l *LocalFile) GetContentAsYAMLNode() (*yaml.Node, error) {
	idx := l.GetIndex()
	if idx != nil && idx.root != nil {
		return idx.GetRootNode(), nil
	}

	// Lock before proceeding with parsing or modifications
	l.parseMutex.Lock()
	defer l.parseMutex.Unlock()

	// Check again after locking in case another goroutine completed parsing
	// while we were waiting for the lock
	if l.parsed != nil {
		return l.parsed, nil
	}

	if l.data == nil {
		return nil, fmt.Errorf("no data to parse for file: %s", l.fullPath)
	}
	var root yaml.Node
	err := yaml.Unmarshal(l.data, &root)
	if err != nil {
		// we can't parse it, so create a fake document node with a single string content
		root = yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: string(l.data),
				},
			},
		}
	}
	if idx != nil && idx.root == nil {
		idx.root = &root
	}
	l.parsed = &root
	return &root, err
}

// GetFileExtension returns the FileExtension of the file.
func (l *LocalFile) GetFileExtension() FileExtension {
	return l.extension
}

// GetFullPath returns the full path of the file.
func (l *LocalFile) GetFullPath() string {
	return l.fullPath
}

// GetErrors returns any errors that occurred during the indexing process.
func (l *LocalFile) GetErrors() []error {
	return l.readingErrors
}

// FullPath returns the full path of the file.
func (l *LocalFile) FullPath() string {
	return l.fullPath
}

// Name returns the name of the file.
func (l *LocalFile) Name() string {
	return l.name
}

// Size returns the size of the file.
func (l *LocalFile) Size() int64 {
	return int64(len(l.data))
}

// Mode returns the file mode bits for the file.
func (l *LocalFile) Mode() fs.FileMode {
	return fs.FileMode(0)
}

// ModTime returns the modification time of the file.
func (l *LocalFile) ModTime() time.Time {
	return l.lastModified
}

// IsDir returns true if the file is a directory, it always returns false
func (l *LocalFile) IsDir() bool {
	return false
}

// Sys returns the underlying data source (always returns nil)
func (l *LocalFile) Sys() interface{} {
	return nil
}

// Close closes the file (doesn't do anything, returns no error)
func (l *LocalFile) Close() error {
	return nil
}

// Stat returns the FileInfo for the file.
func (l *LocalFile) Stat() (fs.FileInfo, error) {
	return l, nil
}

// Read reads the file into a byte slice, makes it compatible with io.Reader.
func (l *LocalFile) Read(b []byte) (int, error) {
	if l.offset >= int64(len(l.GetContent())) {
		return 0, io.EOF
	}
	if l.offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: l.GetFullPath(), Err: fs.ErrInvalid}
	}
	n := copy(b, l.GetContent()[l.offset:])
	l.offset += int64(n)
	return n, nil
}

// LocalFSConfig is the configuration for the LocalFS.
type LocalFSConfig struct {
	// the base directory to index
	BaseDirectory string

	// supply your own logger
	Logger *slog.Logger

	// supply a list of specific files to index only
	FileFilters []string

	// supply a custom fs.FS to use
	DirFS fs.FS

	// supply an index configuration to use
	IndexConfig *SpecIndexConfig
}

// NewLocalFSWithConfig creates a new LocalFS with the supplied configuration.
func NewLocalFSWithConfig(config *LocalFSConfig) (*LocalFS, error) {
	var allErrors []error

	log := config.Logger
	if log == nil {
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))
	}

	// if the basedir is an absolute file, we're just going to index that file.
	ext := filepath.Ext(config.BaseDirectory)
	file := filepath.Base(config.BaseDirectory)

	var absBaseDir string
	absBaseDir, _ = filepath.Abs(config.BaseDirectory)

	localFS := &LocalFS{
		indexConfig:         config.IndexConfig,
		fsConfig:            config,
		logger:              log,
		baseDirectory:       absBaseDir,
		entryPointDirectory: config.BaseDirectory,
	}

	// if a directory filesystem is supplied, use that to walk the directory and pick up everything it finds.
	if config.DirFS != nil {
		walkErr := fs.WalkDir(config.DirFS, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// we don't care about directories, or errors, just read everything we can.
			if d.IsDir() {
				if d.Name() != config.BaseDirectory {
					return nil
				}
			}
			if len(ext) > 2 && p != file {
				return nil
			}
			if strings.HasPrefix(p, ".") {
				return nil
			}
			if len(config.FileFilters) > 0 {
				if !slices.Contains(config.FileFilters, p) {
					return nil
				}
			}
			_, fErr := localFS.extractFile(p)
			return fErr
		})

		if walkErr != nil {
			return nil, walkErr
		}
	}

	localFS.readingErrors = allErrors
	return localFS, nil
}

func (l *LocalFS) extractFile(p string) (*LocalFile, error) {
	extension := ExtractFileType(p)
	var readingErrors []error
	abs := p
	config := l.fsConfig
	if !filepath.IsAbs(p) {
		if config != nil && config.BaseDirectory != "" {
			abs, _ = filepath.Abs(filepath.Join(config.BaseDirectory, p))
		} else {
			abs, _ = filepath.Abs(p)
		}
	}
	var fileData []byte
	switch extension {
	case YAML, JSON, JS, GO, TS, CS, C, CPP, PHP, PY, HTML, MD, JAVA, RS, ZIG, RB:
		var file fs.File
		var fileError error
		if config != nil && config.DirFS != nil {
			l.logger.Debug("[rolodex file loader]: collecting file from dirFS", "file", extension, "location", abs)
			file, _ = config.DirFS.Open(p)
		} else {
			l.logger.Debug("[rolodex file loader]: reading local file from OS", "file", extension, "location", abs)
			file, fileError = os.Open(abs)
		}

		// if reading without a directory FS, error out on any error, do not continue.
		if fileError != nil {
			return nil, fileError
		}

		modTime := time.Now()
		stat, _ := file.Stat()
		if stat != nil {
			modTime = stat.ModTime()
		}
		fileData, _ = io.ReadAll(file)

		lf := &LocalFile{
			filename:      p,
			name:          filepath.Base(p),
			extension:     ExtractFileType(p),
			data:          fileData,
			fullPath:      abs,
			lastModified:  modTime,
			readingErrors: readingErrors,
		}
		l.Files.Store(abs, lf)
		return lf, nil
	case UNSUPPORTED:
		if config != nil && config.DirFS != nil {
			l.logger.Warn("[rolodex file loader]: skipping non JSON/YAML file", "file", abs)
		}
	}
	return nil, nil
}

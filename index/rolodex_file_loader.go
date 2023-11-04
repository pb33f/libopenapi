// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// LocalFS is a file system that indexes local files.
type LocalFS struct {
	indexConfig         *SpecIndexConfig
	entryPointDirectory string
	baseDirectory       string
	Files               map[string]RolodexFile
	logger              *slog.Logger
	readingErrors       []error
}

// GetFiles returns the files that have been indexed. A map of RolodexFile objects keyed by the full path of the file.
func (l *LocalFS) GetFiles() map[string]RolodexFile {
	return l.Files
}

// GetErrors returns any errors that occurred during the indexing process.
func (l *LocalFS) GetErrors() []error {
	return l.readingErrors
}

// Open opens a file, returning it or an error. If the file is not found, the error is of type *PathError.
func (l *LocalFS) Open(name string) (fs.File, error) {

	if l.indexConfig != nil && !l.indexConfig.AllowFileLookup {
		return nil, &fs.PathError{Op: "open", Path: name,
			Err: fmt.Errorf("file lookup for '%s' not allowed, set the index configuration "+
				"to AllowFileLookup to be true", name)}
	}

	if !filepath.IsAbs(name) {
		name, _ = filepath.Abs(filepath.Join(l.baseDirectory, name))
	}

	if f, ok := l.Files[name]; ok {
		return f.(*LocalFile), nil
	} else {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
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
	index         *SpecIndex
	parsed        *yaml.Node
	offset        int64
}

// GetIndex returns the *SpecIndex for the file.
func (l *LocalFile) GetIndex() *SpecIndex {
	return l.index
}

// Index returns the *SpecIndex for the file. If the index has not been created, it will be created (indexed)
func (l *LocalFile) Index(config *SpecIndexConfig) (*SpecIndex, error) {
	if l.index != nil {
		return l.index, nil
	}
	content := l.data

	// first, we must parse the content of the file
	info, err := datamodel.ExtractSpecInfoWithDocumentCheck(content, true)
	if err != nil {
		return nil, err
	}

	index := NewSpecIndexWithConfig(info.RootNode, config)
	index.specAbsolutePath = l.fullPath

	l.index = index
	return index, nil

}

// GetContent returns the content of the file as a string.
func (l *LocalFile) GetContent() string {
	return string(l.data)
}

// GetContentAsYAMLNode returns the content of the file as a *yaml.Node. If something went wrong
// then an error is returned.
func (l *LocalFile) GetContentAsYAMLNode() (*yaml.Node, error) {
	if l.parsed != nil {
		return l.parsed, nil
	}
	if l.index != nil && l.index.root != nil {
		return l.index.root, nil
	}
	if l.data == nil {
		return nil, fmt.Errorf("no data to parse for file: %s", l.fullPath)
	}
	var root yaml.Node
	err := yaml.Unmarshal(l.data, &root)
	if err != nil {
		return nil, err
	}
	if l.index != nil && l.index.root == nil {
		l.index.root = &root
	}
	l.parsed = &root
	return &root, nil
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
}

// NewLocalFSWithConfig creates a new LocalFS with the supplied configuration.
func NewLocalFSWithConfig(config *LocalFSConfig) (*LocalFS, error) {
	localFiles := make(map[string]RolodexFile)
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

	walkErr := fs.WalkDir(config.DirFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// we don't care about directories, or errors, just read everything we can.
		if d.IsDir() {
			return nil
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

		extension := ExtractFileType(p)
		var readingErrors []error
		abs, _ := filepath.Abs(filepath.Join(config.BaseDirectory, p))

		var fileData []byte

		switch extension {
		case YAML, JSON:

			dirFile, _ := config.DirFS.Open(p)
			modTime := time.Now()
			stat, _ := dirFile.Stat()
			if stat != nil {
				modTime = stat.ModTime()
			}
			fileData, _ = io.ReadAll(dirFile)
			log.Debug("collecting JSON/YAML file", "file", abs)
			localFiles[abs] = &LocalFile{
				filename:      p,
				name:          filepath.Base(p),
				extension:     ExtractFileType(p),
				data:          fileData,
				fullPath:      abs,
				lastModified:  modTime,
				readingErrors: readingErrors,
			}
		case UNSUPPORTED:
			log.Debug("skipping non JSON/YAML file", "file", abs)
		}
		return nil
	})

	if walkErr != nil {
		return nil, walkErr
	}

	return &LocalFS{
		Files:               localFiles,
		logger:              log,
		baseDirectory:       absBaseDir,
		entryPointDirectory: config.BaseDirectory,
		readingErrors:       allErrors,
	}, nil
}

// NewLocalFS creates a new LocalFS with the supplied base directory.
func NewLocalFS(baseDir string, dirFS fs.FS) (*LocalFS, error) {
	config := &LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         dirFS,
	}
	return NewLocalFSWithConfig(config)
}

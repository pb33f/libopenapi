// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"errors"
	"github.com/pb33f/libopenapi/datamodel"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type CanBeIndexed interface {
	Index(config *SpecIndexConfig) (*SpecIndex, error)
	GetIndex() *SpecIndex
}

type RolodexFile interface {
	GetContent() string
	GetFileExtension() FileExtension
	GetFullPath() string
	GetErrors() []error
	GetContentAsYAMLNode() (*yaml.Node, error)
	Name() string
	ModTime() time.Time
	IsDir() bool
	Sys() any
	Size() int64
	Mode() os.FileMode
}

var logger *slog.Logger

func init() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

}

type RolodexFS interface {
	Open(name string) (fs.File, error)
	GetFiles() map[string]RolodexFile
}

type Rolodex struct {
	localFS          map[string]fs.FS
	remoteFS         map[string]fs.FS
	indexed          bool
	built            bool
	resolved         bool
	circChecked      bool
	indexConfig      *SpecIndexConfig
	indexingDuration time.Duration
	indexes          []*SpecIndex
	rootIndex        *SpecIndex
	caughtErrors     []error
}

type rolodexFile struct {
	location   string
	rolodex    *Rolodex
	index      *SpecIndex
	localFile  *LocalFile
	remoteFile *RemoteFile
}

func (rf *rolodexFile) Name() string {
	if rf.localFile != nil {
		return rf.localFile.filename
	}
	if rf.remoteFile != nil {
		return rf.remoteFile.filename
	}
	return ""
}

func (rf *rolodexFile) GetIndex() *SpecIndex {
	return rf.index
}

func (rf *rolodexFile) Index(config *SpecIndexConfig) (*SpecIndex, error) {
	if rf.index != nil {
		return rf.index, nil
	}
	var content []byte
	if rf.localFile != nil {
		content = rf.localFile.data
	}
	if rf.remoteFile != nil {
		content = rf.remoteFile.data
	}

	// first, we must parse the content of the file
	info, err := datamodel.ExtractSpecInfo(content)
	if err != nil {
		return nil, err
	}

	// create a new index for this file and link it to this rolodex.
	config.Rolodex = rf.rolodex
	index := NewSpecIndexWithConfig(info.RootNode, config)
	rf.index = index
	return index, nil

}

func (rf *rolodexFile) GetContent() string {
	if rf.localFile != nil {
		return string(rf.localFile.data)
	}
	if rf.remoteFile != nil {
		return string(rf.remoteFile.data)
	}
	return ""
}

func (rf *rolodexFile) GetContentAsYAMLNode() (*yaml.Node, error) {
	if rf.localFile != nil {
		return rf.localFile.GetContentAsYAMLNode()
	}
	if rf.remoteFile != nil {
		return rf.remoteFile.GetContentAsYAMLNode()
	}
	return nil, nil
}

func (rf *rolodexFile) GetFileExtension() FileExtension {
	if rf.localFile != nil {
		return rf.localFile.extension
	}
	if rf.remoteFile != nil {
		return rf.remoteFile.extension
	}
	return UNSUPPORTED
}
func (rf *rolodexFile) GetFullPath() string {
	if rf.localFile != nil {
		return rf.localFile.fullPath
	}
	if rf.remoteFile != nil {
		return rf.remoteFile.fullPath
	}
	return ""
}
func (rf *rolodexFile) ModTime() time.Time {
	if rf.localFile != nil {
		return rf.localFile.lastModified
	}
	if rf.remoteFile != nil {
		return rf.remoteFile.lastModified
	}
	return time.Time{}
}

func (rf *rolodexFile) Size() int64 {
	if rf.localFile != nil {
		return rf.localFile.Size()
	}
	if rf.remoteFile != nil {
		return rf.remoteFile.Size()
	}
	return 0
}

func (rf *rolodexFile) IsDir() bool {
	return false
}

func (rf *rolodexFile) Sys() interface{} {
	return nil
}

func (rf *rolodexFile) Mode() os.FileMode {
	if rf.localFile != nil {
		return rf.localFile.Mode()
	}
	if rf.remoteFile != nil {
		return rf.remoteFile.Mode()
	}
	return os.FileMode(0)
}

func (rf *rolodexFile) GetErrors() []error {
	if rf.localFile != nil {
		return rf.localFile.readingErrors
	}
	if rf.remoteFile != nil {
		return rf.remoteFile.seekingErrors
	}
	return nil
}

func NewRolodex(indexConfig *SpecIndexConfig) *Rolodex {

	r := &Rolodex{
		indexConfig: indexConfig,
		localFS:     make(map[string]fs.FS),
		remoteFS:    make(map[string]fs.FS),
	}
	indexConfig.Rolodex = r
	return r
}

func (r *Rolodex) GetIndexingDuration() time.Duration {
	return r.indexingDuration
}

func (r *Rolodex) GetRootIndex() *SpecIndex {
	return r.rootIndex
}

func (r *Rolodex) GetIndexes() []*SpecIndex {
	return r.indexes
}

func (r *Rolodex) GetCaughtErrors() []error {
	return r.caughtErrors
}

func (r *Rolodex) AddLocalFS(baseDir string, fileSystem fs.FS) {
	absBaseDir, _ := filepath.Abs(baseDir)
	r.localFS[absBaseDir] = fileSystem
}

func (r *Rolodex) AddRemoteFS(baseURL string, fileSystem fs.FS) {
	r.remoteFS[baseURL] = fileSystem
}

func (r *Rolodex) IndexTheRolodex() error {
	if r.indexed {
		return nil
	}

	// disable index building, it will need to be run after the rolodex indexed
	// at a high level.
	r.indexConfig.AvoidBuildIndex = true

	var caughtErrors []error

	var indexBuildQueue []*SpecIndex

	indexRolodexFile := func(
		location string, fs fs.FS,
		doneChan chan bool,
		errChan chan error,
		indexChan chan *SpecIndex) {

		var wg sync.WaitGroup

		indexFileFunc := func(idxFile CanBeIndexed, fullPath string) {
			defer wg.Done()

			// copy config and set the
			copiedConfig := *r.indexConfig
			copiedConfig.SpecAbsolutePath = fullPath
			copiedConfig.AvoidBuildIndex = true // we will build out everything in two steps.
			idx, err := idxFile.Index(&copiedConfig)

			// for each index, we need a resolver
			resolver := NewResolver(idx)
			idx.resolver = resolver

			// check if the config has been set to ignore circular references in arrays and polymorphic schemas
			if copiedConfig.IgnoreArrayCircularReferences {
				resolver.IgnoreArrayCircularReferences()
			}
			if copiedConfig.IgnorePolymorphicCircularReferences {
				resolver.IgnorePolymorphicCircularReferences()
			}
			resolvingErrors := resolver.CheckForCircularReferences()
			for e := range resolvingErrors {
				caughtErrors = append(caughtErrors, resolvingErrors[e])
			}

			if err != nil {
				errChan <- err
			}
			indexChan <- idx
		}

		if lfs, ok := fs.(*LocalFS); ok {
			for _, f := range lfs.Files {
				if idxFile, ko := f.(CanBeIndexed); ko {
					wg.Add(1)
					go indexFileFunc(idxFile, f.GetFullPath())
				}
			}
			wg.Wait()
			doneChan <- true
			return
		}
	}

	indexingCompleted := 0
	totalToIndex := len(r.localFS) + len(r.remoteFS)
	doneChan := make(chan bool)
	errChan := make(chan error)
	indexChan := make(chan *SpecIndex)

	// run through every file system and index every file, fan out as many goroutines as possible.
	started := time.Now()
	for k, v := range r.localFS {
		go indexRolodexFile(k, v, doneChan, errChan, indexChan)
	}
	for k, v := range r.remoteFS {
		go indexRolodexFile(k, v, doneChan, errChan, indexChan)
	}

	for indexingCompleted < totalToIndex {
		select {
		case <-doneChan:
			indexingCompleted++
		case err := <-errChan:
			indexingCompleted++
			caughtErrors = append(caughtErrors, err)
		case idx := <-indexChan:
			indexBuildQueue = append(indexBuildQueue, idx)
		}
	}

	// now that we have indexed all the files, we can build the index.
	r.indexes = indexBuildQueue
	if !r.indexConfig.AvoidBuildIndex {
		for _, idx := range indexBuildQueue {
			idx.BuildIndex()
		}
	}

	// indexed and built every supporting file, we can build the root index (our entry point)
	index := NewSpecIndexWithConfig(r.indexConfig.SpecInfo.RootNode, r.indexConfig)
	resolver := NewResolver(index)
	if r.indexConfig.IgnoreArrayCircularReferences {
		resolver.IgnoreArrayCircularReferences()
	}
	if r.indexConfig.IgnorePolymorphicCircularReferences {
		resolver.IgnorePolymorphicCircularReferences()
	}

	if !r.indexConfig.AvoidBuildIndex {
		index.BuildIndex()
	}

	if !r.indexConfig.AvoidCircularReferenceCheck {
		resolvingErrors := resolver.CheckForCircularReferences()
		for e := range resolvingErrors {
			caughtErrors = append(caughtErrors, resolvingErrors[e])
		}
	}

	r.rootIndex = index
	r.indexingDuration = time.Now().Sub(started)
	r.indexed = true
	r.caughtErrors = caughtErrors
	return errors.Join(caughtErrors...)

}

func (r *Rolodex) CheckForCircularReferences() {
	if r.rootIndex != nil && r.rootIndex.resolver != nil {
		resolvingErrors := r.rootIndex.resolver.CheckForCircularReferences()
		for e := range resolvingErrors {
			r.caughtErrors = append(r.caughtErrors, resolvingErrors[e])
		}
	}
}

func (r *Rolodex) BuildIndexes() {
	if r.built {
		return
	}
	for _, idx := range r.indexes {
		idx.BuildIndex()
	}
	if r.rootIndex != nil {
		r.rootIndex.BuildIndex()
	}
	r.built = true
	return
}

func (r *Rolodex) Open(location string) (RolodexFile, error) {

	var errorStack []error

	var localFile *LocalFile
	//var remoteFile *RemoteFile

	if r == nil || r.localFS == nil && r.remoteFS == nil {
		panic("WHAT NO....")
	}

	for k, v := range r.localFS {

		// check if this is a URL or an abs/rel reference.
		fileLookup := location
		isUrl := false
		u, _ := url.Parse(location)
		if u != nil && u.Scheme != "" {
			isUrl = true
		}

		// TODO handle URLs.
		if !isUrl {
			if !filepath.IsAbs(location) {
				fileLookup, _ = filepath.Abs(filepath.Join(k, location))
			}

			f, err := v.Open(fileLookup)
			if err != nil {

				// try a lookup that is not absolute, but relative
				f, err = v.Open(location)
				if err != nil {
					errorStack = append(errorStack, err)
					continue
				}
			}
			// check if this is a native rolodex FS, then the work is done.
			if lrf, ok := interface{}(f).(*localRolodexFile); ok {

				if lf, ko := interface{}(lrf.f).(*LocalFile); ko {
					localFile = lf
					break
				}
			} else {
				// not a native FS, so we need to read the file and create a local file.
				bytes, rErr := io.ReadAll(f)
				if rErr != nil {
					errorStack = append(errorStack, rErr)
					continue
				}
				s, sErr := f.Stat()
				if sErr != nil {
					errorStack = append(errorStack, sErr)
					continue
				}
				if len(bytes) > 0 {
					localFile = &LocalFile{
						filename:     filepath.Base(fileLookup),
						name:         filepath.Base(fileLookup),
						extension:    ExtractFileType(fileLookup),
						data:         bytes,
						fullPath:     fileLookup,
						lastModified: s.ModTime(),
						index:        r.rootIndex,
					}
					break
				}
			}
		}
	}
	if localFile != nil {
		return &rolodexFile{
			rolodex:   r,
			location:  localFile.fullPath,
			localFile: localFile,
		}, errors.Join(errorStack...)
	}

	return nil, errors.Join(errorStack...)
}

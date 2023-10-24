// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"errors"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel"
	"log/slog"
	"runtime"

	"golang.org/x/sync/syncmap"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type RemoteURLHandler = func(url string) (*http.Response, error)

type RemoteFS struct {
	indexConfig       *SpecIndexConfig
	rootURL           string
	rootURLParsed     *url.URL
	RemoteHandlerFunc RemoteURLHandler
	Files             syncmap.Map
	ProcessingFiles   syncmap.Map
	FetchTime         int64
	FetchChannel      chan *RemoteFile
	remoteErrors      []error
	logger            *slog.Logger
	defaultClient     *http.Client
	extractedFiles    map[string]RolodexFile
}

type RemoteFile struct {
	filename      string
	name          string
	extension     FileExtension
	data          []byte
	fullPath      string
	URL           *url.URL
	lastModified  time.Time
	seekingErrors []error
	index         *SpecIndex
	parsed        *yaml.Node
	offset        int64
}

func (f *RemoteFile) GetFileName() string {
	return f.filename
}

func (f *RemoteFile) GetContent() string {
	return string(f.data)
}

func (f *RemoteFile) GetContentAsYAMLNode() (*yaml.Node, error) {
	if f.parsed != nil {
		return f.parsed, nil
	}
	if f.index != nil && f.index.root != nil {
		return f.index.root, nil
	}
	if f.data == nil {
		return nil, fmt.Errorf("no data to parse for file: %s", f.fullPath)
	}
	var root yaml.Node
	err := yaml.Unmarshal(f.data, &root)
	if err != nil {
		return nil, err
	}
	if f.index != nil && f.index.root == nil {
		f.index.root = &root
	}
	f.parsed = &root
	return &root, nil
}

func (f *RemoteFile) GetFileExtension() FileExtension {
	return f.extension
}

func (f *RemoteFile) GetLastModified() time.Time {
	return f.lastModified
}

func (f *RemoteFile) GetErrors() []error {
	return f.seekingErrors
}

func (f *RemoteFile) GetFullPath() string {
	return f.fullPath
}

// fs.FileInfo interfaces

func (f *RemoteFile) Name() string {
	return f.name
}

func (f *RemoteFile) Size() int64 {
	return int64(len(f.data))
}

func (f *RemoteFile) Mode() fs.FileMode {
	return fs.FileMode(0)
}

func (f *RemoteFile) ModTime() time.Time {
	return f.lastModified
}

func (f *RemoteFile) IsDir() bool {
	return false
}

// fs.File interfaces

func (f *RemoteFile) Sys() interface{} {
	return nil
}

func (f *RemoteFile) Close() error {
	return nil
}
func (f *RemoteFile) Stat() (fs.FileInfo, error) {
	return f, nil
}
func (f *RemoteFile) Read(b []byte) (int, error) {
	if f.offset >= int64(len(f.data)) {
		return 0, io.EOF
	}
	if f.offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: fs.ErrInvalid}
	}
	n := copy(b, f.data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *RemoteFile) Index(config *SpecIndexConfig) (*SpecIndex, error) {

	if f.index != nil {
		return f.index, nil
	}
	content := f.data

	// first, we must parse the content of the file
	info, err := datamodel.ExtractSpecInfoWithDocumentCheck(content, true)
	if err != nil {
		return nil, err
	}

	index := NewSpecIndexWithConfig(info.RootNode, config)

	index.specAbsolutePath = config.SpecAbsolutePath
	f.index = index
	return index, nil
}
func (f *RemoteFile) GetIndex() *SpecIndex {
	return f.index
}

type FileExtension int

const (
	YAML FileExtension = iota
	JSON
	UNSUPPORTED
)

func NewRemoteFSWithConfig(specIndexConfig *SpecIndexConfig) (*RemoteFS, error) {
	remoteRootURL := specIndexConfig.BaseURL
	log := specIndexConfig.Logger
	if log == nil {
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))
	}

	rfs := &RemoteFS{
		indexConfig:   specIndexConfig,
		logger:        log,
		rootURLParsed: remoteRootURL,
		FetchChannel:  make(chan *RemoteFile),
	}
	if remoteRootURL != nil {
		rfs.rootURL = remoteRootURL.String()
	}
	if specIndexConfig.RemoteURLHandler != nil {
		rfs.RemoteHandlerFunc = specIndexConfig.RemoteURLHandler
	} else {
		// default http client
		client := &http.Client{
			Timeout: time.Second * 60,
		}
		rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
			return client.Get(url)
		}
	}
	return rfs, nil
}

func NewRemoteFSWithRootURL(rootURL string) (*RemoteFS, error) {
	remoteRootURL, err := url.Parse(rootURL)
	if err != nil {
		return nil, err
	}
	config := CreateOpenAPIIndexConfig()
	config.BaseURL = remoteRootURL
	return NewRemoteFSWithConfig(config)
}

func (i *RemoteFS) SetRemoteHandlerFunc(handlerFunc RemoteURLHandler) {
	i.RemoteHandlerFunc = handlerFunc
}

func (i *RemoteFS) SetIndexConfig(config *SpecIndexConfig) {
	i.indexConfig = config
}

func (i *RemoteFS) GetFiles() map[string]RolodexFile {
	files := make(map[string]RolodexFile)
	i.Files.Range(func(key, value interface{}) bool {
		files[key.(string)] = value.(*RemoteFile)
		return true
	})
	i.extractedFiles = files
	return files
}

func (i *RemoteFS) GetErrors() []error {
	return i.remoteErrors
}

func (i *RemoteFS) Open(remoteURL string) (fs.File, error) {

	if i.indexConfig != nil && !i.indexConfig.AllowRemoteLookup {
		return nil, fmt.Errorf("remote lookup for '%s' is not allowed, please set "+
			"AllowRemoteLookup to true as part of the index configuration", remoteURL)
	}

	remoteParsedURL, err := url.Parse(remoteURL)
	if err != nil {
		return nil, err
	}
	remoteParsedURLOriginal, _ := url.Parse(remoteURL)

	// try path first
	if r, ok := i.Files.Load(remoteParsedURL.Path); ok {
		return r.(*RemoteFile), nil
	}

	// if we're processing, we need to block and wait for the file to be processed
	// try path first
	if _, ok := i.ProcessingFiles.Load(remoteParsedURL.Path); ok {
		// we can't block if we only have a single CPU, as we'll deadlock, only when we're running in parallel
		// can we block threads.
		if runtime.GOMAXPROCS(-1) > 1 {
			i.logger.Debug("waiting for existing fetch to complete", "file", remoteURL, "remoteURL", remoteParsedURL.String())
			for {
				if wf, ko := i.Files.Load(remoteParsedURL.Path); ko {
					return wf.(*RemoteFile), nil
				}
			}
		}
	}

	// add to processing
	i.ProcessingFiles.Store(remoteParsedURL.Path, true)

	fileExt := ExtractFileType(remoteParsedURL.Path)

	if fileExt == UNSUPPORTED {
		return nil, &fs.PathError{Op: "open", Path: remoteURL, Err: fs.ErrInvalid}
	}

	// if the remote URL is absolute (http:// or https://), and we have a rootURL defined, we need to override
	// the host being defined by this URL, and use the rootURL instead, but keep the path.
	if i.rootURLParsed != nil {
		remoteParsedURL.Host = i.rootURLParsed.Host
		remoteParsedURL.Scheme = i.rootURLParsed.Scheme
		if !filepath.IsAbs(remoteParsedURL.Path) {
			remoteParsedURL.Path = filepath.Join(i.rootURLParsed.Path, remoteParsedURL.Path)
		}
	}

	i.logger.Debug("loading remote file", "file", remoteURL, "remoteURL", remoteParsedURL.String())

	//// no handler func? use the default client.
	//if i.RemoteHandlerFunc == nil {
	//	i.RemoteHandlerFunc = i.defaultClient.Get
	//}

	response, clientErr := i.RemoteHandlerFunc(remoteParsedURL.String())
	if clientErr != nil {

		i.remoteErrors = append(i.remoteErrors, clientErr)
		// remove from processing
		i.ProcessingFiles.Delete(remoteParsedURL.Path)
		if response != nil {
			i.logger.Error("client error", "error", clientErr, "status", response.StatusCode)
		} else {
			i.logger.Error("client error, empty body", "error", clientErr.Error())
		}
		return nil, clientErr
	}

	responseBytes, readError := io.ReadAll(response.Body)
	if readError != nil {

		// remove from processing
		i.ProcessingFiles.Delete(remoteParsedURL.Path)

		return nil, readError
	}

	if response.StatusCode >= 400 {

		// remove from processing
		i.ProcessingFiles.Delete(remoteParsedURL.Path)

		i.logger.Error("unable to fetch remote document",
			"file", remoteParsedURL.Path, "status", response.StatusCode, "resp", string(responseBytes))
		return nil, fmt.Errorf("unable to fetch remote document: %s", string(responseBytes))
	}

	absolutePath, pathErr := filepath.Abs(remoteParsedURL.Path)
	if pathErr != nil {
		// remove from processing
		i.ProcessingFiles.Delete(remoteParsedURL.Path)
		return nil, pathErr
	}

	// extract last modified from response
	lastModified := response.Header.Get("Last-Modified")

	// parse the last modified date into a time object
	lastModifiedTime, parseErr := time.Parse(time.RFC1123, lastModified)

	if parseErr != nil {
		// can't extract last modified, so use now
		lastModifiedTime = time.Now()
	}

	filename := filepath.Base(remoteParsedURL.Path)

	remoteFile := &RemoteFile{
		filename:     filename,
		name:         remoteParsedURL.Path,
		extension:    fileExt,
		data:         responseBytes,
		fullPath:     absolutePath,
		URL:          remoteParsedURL,
		lastModified: lastModifiedTime,
	}

	copiedCfg := *i.indexConfig

	newBase := fmt.Sprintf("%s://%s%s", remoteParsedURLOriginal.Scheme, remoteParsedURLOriginal.Host,
		filepath.Dir(remoteParsedURL.Path))
	newBaseURL, _ := url.Parse(newBase)

	if newBaseURL != nil {
		copiedCfg.BaseURL = newBaseURL
	}
	copiedCfg.SpecAbsolutePath = remoteParsedURL.String()
	idx, idxError := remoteFile.Index(&copiedCfg)

	if len(remoteFile.data) > 0 {
		i.logger.Debug("successfully loaded file", "file", absolutePath)
	}
	//i.seekRelatives(remoteFile)

	if idxError != nil && idx == nil {
		i.remoteErrors = append(i.remoteErrors, idxError)
	} else {

		// for each index, we need a resolver
		resolver := NewResolver(idx)
		idx.resolver = resolver
		idx.BuildIndex()
	}

	// remove from processing
	i.ProcessingFiles.Delete(remoteParsedURL.Path)
	i.Files.Store(absolutePath, remoteFile)

	//if !i.remoteRunning {
	return remoteFile, errors.Join(i.remoteErrors...)
	//	} else {
	//		return remoteFile, nil/
	//	}
}

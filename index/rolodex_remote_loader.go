// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"errors"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/syncmap"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type RemoteFS struct {
	indexConfig       *SpecIndexConfig
	rootURL           string
	rootURLParsed     *url.URL
	RemoteHandlerFunc RemoteURLHandler
	Files             syncmap.Map
	FetchTime         int64
	FetchChannel      chan *RemoteFile
	remoteWg          sync.WaitGroup
	remoteRunning     bool
	remoteErrorLock   sync.Mutex
	remoteErrors      []error
	logger            *slog.Logger
	defaultClient     *http.Client
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
	index.specAbsolutePath = f.fullPath
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

	// TODO: handle logging

	rfs := &RemoteFS{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
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
	return files
}

func (i *RemoteFS) seekRelatives(file *RemoteFile) {

	extractedRefs := ExtractRefs(string(file.data))
	if len(extractedRefs) == 0 {
		return
	}

	fetchChild := func(url string) {
		_, err := i.Open(url)
		if err != nil {
			file.seekingErrors = append(file.seekingErrors, err)
			i.remoteErrorLock.Lock()
			i.remoteErrors = append(i.remoteErrors, err)
			i.remoteErrorLock.Unlock()
		}
		defer i.remoteWg.Done()
	}

	for _, ref := range extractedRefs {
		refType := ExtractRefType(ref[1])
		switch refType {
		case File:
			fileLocation, _ := ExtractRefValues(ref[1])
			//parentDir, _ := filepath.Abs(filepath.Dir(file.fullPath))
			var fullPath string
			if filepath.IsAbs(fileLocation) {
				fullPath = fileLocation
			} else {
				fullPath, _ = filepath.Abs(filepath.Join(filepath.Dir(file.fullPath), fileLocation))
			}

			if f, ok := i.Files.Load(fullPath); ok {
				i.logger.Debug("file already loaded, skipping", "file", f.(*RemoteFile).fullPath)
				continue
			} else {
				i.remoteWg.Add(1)
				go fetchChild(fullPath)
			}

		case HTTP:
			fmt.Printf("Found relative HTTP reference: %s\n", ref[1])
		}
	}
	if !i.remoteRunning {
		i.remoteRunning = true
		i.remoteWg.Wait()
		i.remoteRunning = false
	}

}

func (i *RemoteFS) Open(remoteURL string) (fs.File, error) {

	remoteParsedURL, err := url.Parse(remoteURL)
	if err != nil {
		return nil, err
	}

	// try path first
	if r, ok := i.Files.Load(remoteParsedURL.Path); ok {
		return r.(*RemoteFile), nil
	}

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

	i.logger.Debug("Loading remote file", "file", remoteURL, "remoteURL", remoteParsedURL.String())

	// no handler func? use the default client.
	if i.RemoteHandlerFunc == nil {
		i.RemoteHandlerFunc = i.defaultClient.Get
	}

	response, clientErr := i.RemoteHandlerFunc(remoteParsedURL.String())
	if clientErr != nil {
		if response != nil {
			i.logger.Error("client error", "error", clientErr, "status", response.StatusCode)
		} else {
			i.logger.Error("no response for request", "error", clientErr.Error())
		}
		return nil, clientErr
	}

	responseBytes, readError := io.ReadAll(response.Body)
	if readError != nil {
		return nil, readError
	}

	if response.StatusCode >= 400 {
		i.logger.Error("Unable to fetch remote document",
			"file", remoteParsedURL.Path, "status", response.StatusCode, "resp", string(responseBytes))
		return nil, fmt.Errorf("unable to fetch remote document: %s", string(responseBytes))
	}

	absolutePath, pathErr := filepath.Abs(remoteParsedURL.Path)
	if pathErr != nil {
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

	newBase := fmt.Sprintf("%s://%s%s", remoteParsedURL.Scheme, remoteParsedURL.Host,
		filepath.Dir(remoteParsedURL.Path))
	newBaseURL, _ := url.Parse(newBase)

	copiedCfg.BaseURL = newBaseURL
	copiedCfg.SpecAbsolutePath = remoteParsedURL.String()
	idx, _ := remoteFile.Index(&copiedCfg)

	// for each index, we need a resolver
	resolver := NewResolver(idx)
	idx.resolver = resolver

	i.Files.Store(absolutePath, remoteFile)

	i.logger.Debug("successfully loaded file", "file", absolutePath)
	i.seekRelatives(remoteFile)

	idx.BuildIndex()

	if !i.remoteRunning {
		return remoteFile, errors.Join(i.remoteErrors...)
	} else {
		return remoteFile, nil
	}
}

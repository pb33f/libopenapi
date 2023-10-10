// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rolodex

import (
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/syncmap"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type RemoteURLHandler = func(url string) (*http.Response, error)

type RemoteFS struct {
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
}

type FileExtension int

const (
	YAML FileExtension = iota
	JSON
)

func NewRemoteFS(rootURL string) (*RemoteFS, error) {
	remoteRootURL, err := url.Parse(rootURL)
	if err != nil {
		return nil, err
	}
	return &RemoteFS{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
		rootURL:       rootURL,
		rootURLParsed: remoteRootURL,
		FetchChannel:  make(chan *RemoteFile),
	}, nil
}

func (i *RemoteFS) seekRelatives(file *RemoteFile) {

	extractedRefs := ExtractRefs(file.data)
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
	if i.remoteRunning == false {
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

	var fileExt FileExtension
	switch filepath.Ext(remoteParsedURL.Path) {
	case ".yaml":
		fileExt = YAML
	case ".json":
		fileExt = JSON
	default:
		return nil, &fs.PathError{Op: "open", Path: remoteURL, Err: fs.ErrInvalid}
	}

	i.logger.Debug("Loading remote file", "file", remoteParsedURL.Path)

	response, clientErr := i.RemoteHandlerFunc(i.rootURL + remoteURL)
	if clientErr != nil {
		i.logger.Error("client error", "error", response.StatusCode)

		return nil, clientErr
	}

	responseBytes, readError := io.ReadAll(response.Body)
	if readError != nil {
		return nil, readError
	}

	if response.StatusCode >= 400 {
		i.logger.Error("Unable to fetch remote document %s",
			"file", remoteParsedURL.Path, "status", response.StatusCode, "resp", string(responseBytes))
		return nil, errors.New(fmt.Sprintf("Unable to fetch remote document: %s", string(responseBytes)))
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
		return nil, parseErr
	}

	remoteFile := &RemoteFile{
		name:         remoteParsedURL.Path,
		extension:    fileExt,
		data:         string(responseBytes),
		fullPath:     absolutePath,
		URL:          remoteParsedURL,
		lastModified: lastModifiedTime,
	}
	i.Files.Store(absolutePath, remoteFile)

	i.logger.Debug("successfully loaded file", "file", absolutePath)
	i.seekRelatives(remoteFile)

	if i.remoteRunning == false {
		return &remoteRolodexFile{remoteFile, 0}, errors.Join(i.remoteErrors...)
	} else {
		return &remoteRolodexFile{remoteFile, 0}, nil
	}
}

type RemoteFile struct {
	name          string
	extension     FileExtension
	data          string
	fullPath      string
	URL           *url.URL
	lastModified  time.Time
	seekingErrors []error
}

func (f *RemoteFile) FullPath() string {
	return f.fullPath
}

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

func (f *RemoteFile) Sys() interface{} {
	return nil
}

type remoteRolodexFile struct {
	f      *RemoteFile
	offset int64
}

func (f *remoteRolodexFile) Close() error               { return nil }
func (f *remoteRolodexFile) Stat() (fs.FileInfo, error) { return f.f, nil }
func (f *remoteRolodexFile) Read(b []byte) (int, error) {
	if f.offset >= int64(len(f.f.data)) {
		return 0, io.EOF
	}
	if f.offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: f.f.name, Err: fs.ErrInvalid}
	}
	n := copy(b, f.f.data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

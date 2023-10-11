// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rolodex

import (
	"errors"
	"io"
	"io/fs"
	"net/url"
	"path/filepath"
	"time"
)

type RolodexFile interface {
	GetFileName() string
	GetContent() string
	GetFileExtension() FileExtension
	GetFullPath() string
	GetLastModified() time.Time
	GetErrors() []error
}

type RolodexFS struct {
	fs fs.FS
}

type Rolodex struct {
	localFS  map[string]fs.FS
	remoteFS map[string]fs.FS
}

type rolodexFile struct {
	location   string
	localFile  *LocalFile
	remoteFile *RemoteFile
}

func (rf *rolodexFile) GetFileName() string {
	if rf.localFile != nil {
		return rf.localFile.filename
	}
	if rf.remoteFile != nil {
		return rf.remoteFile.filename
	}
	return ""
}
func (rf *rolodexFile) GetContent() string {
	if rf.localFile != nil {
		return rf.localFile.data
	}
	if rf.remoteFile != nil {
		return rf.remoteFile.data
	}
	return ""
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
func (rf *rolodexFile) GetLastModified() time.Time {
	if rf.localFile != nil {
		return rf.localFile.lastModified
	}
	if rf.remoteFile != nil {
		return rf.remoteFile.lastModified
	}
	return time.Time{}
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

func NewRolodex() *Rolodex {
	return &Rolodex{
		localFS:  make(map[string]fs.FS),
		remoteFS: make(map[string]fs.FS),
	}
}

func (r *Rolodex) AddLocalFS(baseDir string, fileSystem fs.FS) {
	r.localFS[baseDir] = fileSystem
}

func (r *Rolodex) AddRemoteFS(baseURL string, fileSystem fs.FS) {
	r.remoteFS[baseURL] = fileSystem
}

func (r *Rolodex) Open(location string) (RolodexFile, error) {

	var errorStack []error

	var localFile *LocalFile
	//var remoteFile *RemoteFile

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
						data:         string(bytes),
						fullPath:     fileLookup,
						lastModified: s.ModTime(),
					}
					break
				}
			}
		}
	}
	if localFile != nil {
		return &rolodexFile{
			location:  localFile.fullPath,
			localFile: localFile,
		}, errors.Join(errorStack...)
	}

	return nil, errors.Join(errorStack...)
}

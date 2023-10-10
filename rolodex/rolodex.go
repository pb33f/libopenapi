// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rolodex

import (
	"golang.org/x/exp/slices"
	"io/fs"
	"net/http"
	"path/filepath"
)

type Rolodex struct {
	Files  []*RolodexFile
	client *http.Client
}

type RolodexFile struct {
	Path    string
	Content []byte
}

func (rolodex *Rolodex) FindFile(path string) *RolodexFile {
	for _, f := range rolodex.Files {
		if f.Path == path {
			return f
		}
	}
	return nil
}

func (rolodex *Rolodex) ExploreURL(url string) error {
	return nil
}

func Files(root string, fileSystem fs.FS) *Rolodex {

	var files []*RolodexFile
	extensions := []string{".yaml", ".json", ".yml"}
	_ = fs.WalkDir(fileSystem, root, func(p string, d fs.DirEntry, err error) error {
		if slices.Contains(extensions, filepath.Ext(p)) {
			fileData, _ := fs.ReadFile(fileSystem, p)
			files = append(files, &RolodexFile{
				Path:    p,
				Content: fileData,
			})
		}
		return nil
	})
	return &Rolodex{
		Files: files,
	}
}

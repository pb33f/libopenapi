// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"fmt"
	"io/fs"
	"maps"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pb33f/libopenapi/datamodel"
)

// MemberError identifies the collection member that failed to load. Its text is
// the member's own diagnostic, which already names the file.
type MemberError struct {
	Name string
	Err  error
}

func (e *MemberError) Error() string { return e.Err.Error() }

func (e *MemberError) Unwrap() error { return e.Err }

// LoadCollection registers all named schema sources before resolving references.
// Names are slash-separated paths relative to directory. An empty snapshot is valid.
// The returned document has a nil Root and one entry in Sources per physical file.
func LoadCollection(members map[string][]byte, directory string, config *datamodel.DocumentConfiguration) (*Document, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	if config == nil {
		config = datamodel.NewDocumentConfiguration()
	}
	base := fileURL(absolute)
	base.Path = strings.TrimSuffix(base.Path, "/") + "/"
	d := newDocument(config, base)
	for _, name := range slices.Sorted(maps.Keys(members)) {
		if !fs.ValidPath(name) || name == "." || strings.ContainsRune(name, '\\') {
			d.Release()
			return nil, fmt.Errorf("invalid schema member path %q", name)
		}
		uri := base.ResolveReference(&url.URL{Path: name})
		if _, err := d.addSource(members[name], uri, true); err != nil {
			d.Release()
			return nil, &MemberError{Name: name, Err: err}
		}
	}
	if err := d.resolveReferences(); err != nil {
		d.Release()
		return nil, err
	}
	return d, nil
}

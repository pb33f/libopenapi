// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"errors"
	"fmt"

	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pb33f/go-yaml"
)

var anchorName = regexp.MustCompile(`^[A-Za-z_][-A-Za-z0-9._]*$`)

func (d *Document) resolveReferences() error {
	for i := 0; i < len(d.References); i++ {
		ref := d.References[i]
		target, err := d.resolve(ref.address)
		if err != nil {
			return fmt.Errorf("%s:%d: resolve $ref %q: %w", d.schemas[ref.From].Source.Name, ref.From.Line, ref.Value, err)
		}
		d.References[i].To = target
	}
	return nil
}

func (d *Document) resolve(address string) (*yaml.Node, error) {
	if target := d.lookup(address); target != nil {
		return target, nil
	}
	uri, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	fragment := uri.Fragment
	uri.Fragment, uri.RawFragment = "", ""
	root := d.lookup(uri.String())
	if root == nil {
		data, err := d.read(uri)
		if err != nil {
			return nil, err
		}
		resource, err := d.addSource(data, uri, true)
		if err != nil {
			return nil, err
		}
		if target := d.lookup(address); target != nil {
			return target, nil
		}
		root = resource.Node
	}
	target := root
	if fragment != "" {
		if !strings.HasPrefix(fragment, "/") {
			canonical := *d.schemas[root].base
			canonical.Fragment = fragment
			if target := d.lookup(canonical.String()); target != nil {
				return target, nil
			}
			return nil, fmt.Errorf("unknown schema anchor %q", fragment)
		}
		for part := range strings.SplitSeq(fragment[1:], "/") {
			key := strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
			if target.Kind == yaml.SequenceNode {
				i, err := strconv.Atoi(key)
				if err != nil || i < 0 || i >= len(target.Content) || strconv.Itoa(i) != key {
					target = nil
				} else {
					target = target.Content[i]
				}
			} else {
				target = member(target, key)
			}
			if target == nil {
				break
			}
		}
	}
	if d.schemas[target] == nil {
		return nil, fmt.Errorf("no schema at %q", address)
	}
	return target, nil
}

// Read source bytes without attaching a second API-format index to this Rolodex.
// In particular, a revision filesystem controls all local reads for its snapshot.
func (d *Document) read(uri *url.URL) ([]byte, error) {
	switch uri.Scheme {
	case "file":
		if uri.Host != "" && uri.Host != "localhost" {
			return nil, fmt.Errorf("unsupported file URI host %q", uri.Host)
		}
		if !d.config.AllowFileReferences && d.config.BasePath == "" {
			return nil, fmt.Errorf("file references are disabled")
		}
		filename := filepath.FromSlash(uri.Path)
		if d.config.LocalFS == nil {
			return os.ReadFile(filename)
		}
		dir := d.config.BasePath
		if dir == "" {
			dir = filepath.FromSlash(d.base.Path)
		}
		relative, err := filepath.Rel(dir, filename)
		if err != nil {
			return nil, err
		}
		return readSource(d.config.LocalFS, filepath.ToSlash(relative), filename)
	case "http", "https":
		if !d.config.AllowRemoteReferences {
			return nil, fmt.Errorf("remote references are disabled")
		}
		if d.config.RemoteFS != nil {
			return readSource(d.config.RemoteFS, uri.String(), "")
		}
		get := d.config.RemoteURLHandler
		if get == nil {
			get = remoteClient.Get
		}
		res, err := get(uri.String())
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return nil, fmt.Errorf("HTTP %d", res.StatusCode)
		}
		return io.ReadAll(res.Body)
	default:
		return nil, fmt.Errorf("no registered schema for %q", d.sourceName(uri))
	}
}

var remoteClient = &http.Client{Timeout: 120 * time.Second}

func readSource(files fs.FS, name, absolute string) ([]byte, error) {
	file, err := files.Open(name)
	if err != nil && absolute != "" && (errors.Is(err, fs.ErrInvalid) || !fs.ValidPath(name)) {
		file, err = files.Open(absolute)
	}
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, fmt.Errorf("missing file %q", name)
	}
	defer file.Close()
	if source, ok := file.(interface{ GetContent() string }); ok {
		return []byte(source.GetContent()), nil
	}
	return io.ReadAll(file)
}

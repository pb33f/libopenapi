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
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pb33f/go-yaml"
)

var anchorName = regexp.MustCompile(`^[A-Za-z_][-A-Za-z0-9._]*$`)

// File URLs have a leading slash before a Windows drive; native file paths do not.
func fileURL(filename string) *url.URL {
	p := filepath.ToSlash(filename)
	u := &url.URL{Scheme: "file", Path: p}
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(p, "//") {
			host, rest, _ := strings.Cut(p[2:], "/")
			u.Host, u.Path = host, "/"+rest
		} else if filepath.VolumeName(filename) != "" {
			u.Path = "/" + p
		}
	}
	return u
}

func filePath(u *url.URL) string {
	p := u.Path
	if runtime.GOOS == "windows" {
		if u.Host != "" && u.Host != "localhost" {
			p = "//" + u.Host + p
		} else if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
			p = p[1:]
		}
	}
	return filepath.FromSlash(p)
}

func (d *Document) resolveReferences() error {
	for i := 0; i < len(d.References); i++ {
		ref := d.References[i]
		target, err := d.resolve(ref.address, ref.Value)
		if err != nil {
			return fmt.Errorf("%s:%d: resolve %s %q: %w", d.schemas[ref.From].Source.Name, ref.From.Line, ref.Keyword, ref.Value, err)
		}
		if ref.Keyword == "$dynamicRef" {
			u, _ := url.Parse(ref.address)
			if anchor := member(target, "$dynamicAnchor"); anchor != nil && u.Fragment == anchor.Value {
				return fmt.Errorf("%s:%d: dynamic scope resolution for $dynamicRef %q is unsupported", d.schemas[ref.From].Source.Name, ref.From.Line, ref.Value)
			}
		}
		d.References[i].To = target
	}
	return nil
}

func (d *Document) resolve(address, authored string) (*yaml.Node, error) {
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
		retrieval := d.collectionRetrieval(uri, authored)
		root = d.lookup(retrieval.String())
		if root == nil {
			data, err := d.read(retrieval)
			if err != nil {
				return nil, err
			}
			resource, err := d.addSource(data, retrieval, true)
			if err != nil {
				return nil, err
			}
			root = resource.Node
		}
		if retrieval != uri {
			if err := d.register(uri.String(), root, d.schemas[root].Source); err != nil {
				return nil, err
			}
		}
		if target := d.lookup(address); target != nil {
			return target, nil
		}
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
		return nil, fmt.Errorf("no schema at %q in %s", fragment, d.sourceName(uri))
	}
	return target, nil
}

// Collection members are registered under their snapshot paths. An explicit base
// supplies relative dependencies only after those registered locations miss.
func (d *Document) collectionRetrieval(uri *url.URL, authored string) *url.URL {
	if d.Root != nil || uri.Scheme != "file" || d.config.BaseURL == nil && d.config.BasePath == "" {
		return uri
	}
	ref, err := url.Parse(authored)
	if err != nil || ref.IsAbs() || strings.HasPrefix(ref.Path, "/") {
		return uri
	}
	dir := filePath(d.base)
	rel, err := filepath.Rel(dir, filePath(uri))
	if err != nil || !filepath.IsLocal(rel) {
		return uri
	}
	if d.config.BaseURL != nil {
		parent := *d.config.BaseURL
		parent.Path = strings.TrimSuffix(parent.Path, "/") + "/"
		return parent.ResolveReference(&url.URL{Path: filepath.ToSlash(rel), RawQuery: uri.RawQuery})
	}
	if filepath.Clean(dir) == filepath.Clean(d.config.BasePath) {
		return uri
	}
	return fileURL(filepath.Join(d.config.BasePath, rel))
}

// Read source bytes without attaching a second API-format index to this Rolodex.
// In particular, a revision filesystem controls all local reads for its snapshot.
func (d *Document) read(uri *url.URL) ([]byte, error) {
	switch uri.Scheme {
	case "file":
		if uri.Host != "" && uri.Host != "localhost" && (runtime.GOOS != "windows" || d.base.Host != uri.Host) {
			return nil, fmt.Errorf("unsupported file URI host %q", uri.Host)
		}
		if !d.config.AllowFileReferences && d.config.BasePath == "" {
			return nil, fmt.Errorf("file references are disabled")
		}
		filename := filePath(uri)
		if d.config.LocalFS == nil {
			return os.ReadFile(filename)
		}
		dir := d.config.BasePath
		if dir == "" {
			dir = filePath(d.base)
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
			var requestErr *url.Error
			if errors.As(err, &requestErr) {
				return nil, fmt.Errorf("%s: %w", d.sourceName(uri), requestErr.Err)
			}
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

// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"sort"
	"strings"
)

// Paths represents a low-level Swagger / OpenAPI Paths object.
type Paths struct {
	PathItems  map[low.KeyReference[string]]low.ValueReference[*PathItem]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

// GetExtensions returns all Paths extensions and satisfies the low.HasExtensions interface.
func (p *Paths) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return p.Extensions
}

// FindPath attempts to locate a PathItem instance, given a path key.
func (p *Paths) FindPath(path string) *low.ValueReference[*PathItem] {
	for k, j := range p.PathItems {
		if k.Value == path {
			return &j
		}
	}
	return nil
}

// FindPathAndKey attempts to locate a PathItem instance, given a path key.
func (p *Paths) FindPathAndKey(path string) (*low.KeyReference[string], *low.ValueReference[*PathItem]) {
	for k, j := range p.PathItems {
		if k.Value == path {
			return &k, &j
		}
	}
	return nil, nil
}

// FindExtension will attempt to locate an extension value given a name.
func (p *Paths) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, p.Extensions)
}

// Build will extract extensions and paths from node.
func (p *Paths) Build(root *yaml.Node, idx *index.SpecIndex) error {
	p.Extensions = low.ExtractExtensions(root)
	skip := false
	var currentNode *yaml.Node

	pathsMap := make(map[low.KeyReference[string]]low.ValueReference[*PathItem])

	// build each new path, in a new thread.
	type pathBuildResult struct {
		k low.KeyReference[string]
		v low.ValueReference[*PathItem]
	}

	bChan := make(chan pathBuildResult)
	eChan := make(chan error)
	var buildPathItem = func(cNode, pNode *yaml.Node, b chan<- pathBuildResult, e chan<- error) {
		path := new(PathItem)
		_ = low.BuildModel(pNode, path)
		err := path.Build(pNode, idx)
		if err != nil {
			e <- err
			return
		}
		b <- pathBuildResult{
			k: low.KeyReference[string]{
				Value:   cNode.Value,
				KeyNode: cNode,
			},
			v: low.ValueReference[*PathItem]{
				Value:     path,
				ValueNode: pNode,
			},
		}
	}
	pathCount := 0
	for i, pathNode := range root.Content {
		if strings.HasPrefix(strings.ToLower(pathNode.Value), "x-") {
			skip = true
			continue
		}
		if skip {
			skip = false
			continue
		}
		if i%2 == 0 {
			currentNode = pathNode
			continue
		}
		pathCount++
		go buildPathItem(currentNode, pathNode, bChan, eChan)
	}
	completedItems := 0
	for completedItems < pathCount {
		select {
		case err := <-eChan:
			return err
		case res := <-bChan:
			completedItems++
			pathsMap[res.k] = res.v
		}
	}
	p.PathItems = pathsMap
	return nil
}

// Hash will return a consistent SHA256 Hash of the PathItem object
func (p *Paths) Hash() [32]byte {
	var f []string
	l := make([]string, len(p.PathItems))
	keys := make(map[string]low.ValueReference[*PathItem])
	z := 0
	for k := range p.PathItems {
		keys[k.Value] = p.PathItems[k]
		l[z] = k.Value
		z++
	}
	sort.Strings(l)
	for k := range l {
		f = append(f, low.GenerateHashString(keys[l[k]].Value))
	}
	ekeys := make([]string, len(p.Extensions))
	z = 0
	for k := range p.Extensions {
		ekeys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(p.Extensions[k].Value))))
		z++
	}
	sort.Strings(ekeys)
	f = append(f, ekeys...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

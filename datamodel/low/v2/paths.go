// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
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
func (p *Paths) Build(ctx context.Context, _, root *yaml.Node, idx *index.SpecIndex) error {
	root = utils.NodeAlias(root)
	utils.CheckForMergeNodes(root)
	p.Extensions = low.ExtractExtensions(root)

	// Translate YAML nodes to pathsMap using `TranslatePipeline`.
	type pathBuildResult struct {
		key   low.KeyReference[string]
		value low.ValueReference[*PathItem]
	}
	type buildInput struct {
		currentNode *yaml.Node
		pathNode    *yaml.Node
	}
	pathsMap := make(map[low.KeyReference[string]]low.ValueReference[*PathItem])
	in := make(chan buildInput)
	out := make(chan pathBuildResult)
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2) // input and output goroutines.

	// TranslatePipeline input.
	go func() {
		defer func() {
			close(in)
			wg.Done()
		}()
		skip := false
		var currentNode *yaml.Node
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

			select {
			case in <- buildInput{
				currentNode: currentNode,
				pathNode:    pathNode,
			}:
			case <-done:
				return
			}
		}
	}()

	// TranslatePipeline output.
	go func() {
		for {
			result, ok := <-out
			if !ok {
				break
			}
			pathsMap[result.key] = result.value
		}
		close(done)
		wg.Done()
	}()

	translateFunc := func(value buildInput) (pathBuildResult, error) {
		pNode := value.pathNode
		cNode := value.currentNode
		path := new(PathItem)
		_ = low.BuildModel(pNode, path)
		err := path.Build(ctx, cNode, pNode, idx)
		if err != nil {
			return pathBuildResult{}, err
		}
		return pathBuildResult{
			key: low.KeyReference[string]{
				Value:   cNode.Value,
				KeyNode: cNode,
			},
			value: low.ValueReference[*PathItem]{
				Value:     path,
				ValueNode: pNode,
			},
		}, nil
	}
	err := datamodel.TranslatePipeline[buildInput, pathBuildResult](in, out, translateFunc)
	wg.Wait()
	if err != nil {
		return err
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

// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"strings"
)

const (
	PathsLabel   = "paths"
	GetLabel     = "get"
	PostLabel    = "post"
	PatchLabel   = "patch"
	PutLabel     = "put"
	DeleteLabel  = "delete"
	OptionsLabel = "options"
	HeadLabel    = "head"
)

type Paths struct {
	PathItems  map[low.KeyReference[string]]low.ValueReference[*PathItem]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

func (p *Paths) FindPath(path string) *low.ValueReference[*PathItem] {
	for k, j := range p.PathItems {
		if k.Value == path {
			return &j
		}
	}
	return nil
}

func (p *Paths) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, p.Extensions)
}

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

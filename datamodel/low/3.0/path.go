// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
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
    TraceLabel   = "trace"
)

type Paths struct {
    PathItems  map[low.KeyReference[string]]low.ValueReference[*PathItem]
    Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

func (p *Paths) FindPath(path string) *low.ValueReference[*PathItem] {
    for k, p := range p.PathItems {
        if k.Value == path {
            return &p
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

        if ok, _, _ := utils.IsNodeRefValue(pathNode); ok {
            r := low.LocateRefNode(pathNode, idx)
            if r != nil {
                pathNode = r
            } else {
                return fmt.Errorf("path item build failed: cannot find reference: %s at line %d, col %d",
                    pathNode.Content[1].Value, pathNode.Content[1].Line, pathNode.Content[1].Column)
            }
        }

        var path = PathItem{}
        _ = low.BuildModel(pathNode, &path)
        err := path.Build(pathNode, idx)
        if err != nil {
            return err
        }

        // add bulk here
        pathsMap[low.KeyReference[string]{
            Value:   currentNode.Value,
            KeyNode: currentNode,
        }] = low.ValueReference[*PathItem]{
            Value:     &path,
            ValueNode: pathNode,
        }
    }

    p.PathItems = pathsMap
    return nil

}

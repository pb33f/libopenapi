package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
	"sync"
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

func (p *Paths) Build(root *yaml.Node, idx *index.SpecIndex) error {
	p.Extensions = ExtractExtensions(root)
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
		var path = PathItem{}
		err := BuildModel(pathNode, &path)
		if err != nil {

		}
		err = path.Build(pathNode, idx)
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

type PathItem struct {
	Description low.NodeReference[string]
	Summary     low.NodeReference[string]
	Get         low.NodeReference[*Operation]
	Put         low.NodeReference[*Operation]
	Post        low.NodeReference[*Operation]
	Delete      low.NodeReference[*Operation]
	Options     low.NodeReference[*Operation]
	Head        low.NodeReference[*Operation]
	Patch       low.NodeReference[*Operation]
	Trace       low.NodeReference[*Operation]
	Servers     []low.NodeReference[*Server]
	Parameters  []low.NodeReference[*Parameter]
	Extensions  map[low.KeyReference[string]]low.ValueReference[any]
}

func (p *PathItem) Build(root *yaml.Node, idx *index.SpecIndex) error {
	p.Extensions = ExtractExtensions(root)
	skip := false
	var currentNode *yaml.Node

	var wg sync.WaitGroup
	var errors []error

	var ops []low.NodeReference[*Operation]

	if ok, _, _ := utils.IsNodeRefValue(root); ok {
		r := LocateRefNode(root, idx)
		if r != nil {
			root = r
		} else {
			return nil
		}
	}

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

		var op Operation

		wg.Add(1)
		BuildModelAsync(pathNode, &op, &wg, &errors)

		opRef := low.NodeReference[*Operation]{
			Value:     &op,
			KeyNode:   currentNode,
			ValueNode: pathNode,
		}

		ops = append(ops, opRef)

		switch currentNode.Value {
		case GetLabel:
			p.Get = opRef
		case PostLabel:
			p.Post = opRef
		case PutLabel:
			p.Put = opRef
		case PatchLabel:
			p.Patch = opRef
		case DeleteLabel:
			p.Delete = opRef
		case HeadLabel:
			p.Head = opRef
		case OptionsLabel:
			p.Options = opRef
		case TraceLabel:
			p.Trace = opRef
		}
	}

	if len(ops) > 0 {
		//wg.Wait()
	}

	//all operations have been superficially built,
	//now we need to build out the operation, we will do this asynchronously for speed.
	opBuildChan := make(chan bool)
	opErrorChan := make(chan error)

	var buildOpFunc = func(op low.NodeReference[*Operation], ch chan<- bool, errCh chan<- error) {

		//build out the operation.
		er := op.Value.Build(op.ValueNode, idx)
		if er != nil {
			errCh <- er
		}
		ch <- true
	}

	for _, op := range ops {
		go buildOpFunc(op, opBuildChan, opErrorChan)
	}

	n := 0
allDone:
	for {
		select {
		case buildError := <-opErrorChan:
			return buildError
		case <-opBuildChan:
			n++
			if n == len(ops) {
				break allDone
			}

		}
	}

	return nil
}

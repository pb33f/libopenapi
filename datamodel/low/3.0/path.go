package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
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
	Paths      map[low.NodeReference[string]]low.NodeReference[*Path]
	Extensions map[low.NodeReference[string]]low.NodeReference[any]
}

func (p *Paths) GetPathMap() map[string]*Path {
	pMap := make(map[string]*Path)
	for i, pv := range p.Paths {
		pMap[i.Value] = pv.Value
	}
	return pMap
}

func (p *Paths) Build(root *yaml.Node, idx *index.SpecIndex) error {

	// extract extensions
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	p.Extensions = extensionMap
	skip := false
	var currentNode *yaml.Node

	pathsMap := make(map[low.NodeReference[string]]low.NodeReference[*Path])

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
		var path = Path{}
		err = BuildModel(pathNode, &path)
		if err != nil {

		}
		err = path.Build(pathNode, idx)
		if err != nil {
			return err
		}

		// add bulk here
		pathsMap[low.NodeReference[string]{
			Value:   currentNode.Value,
			KeyNode: currentNode,
		}] = low.NodeReference[*Path]{
			Value:     &path,
			ValueNode: pathNode,
		}
	}

	p.Paths = pathsMap
	return nil

}

type Path struct {
	Description low.NodeReference[string]
	Summary     low.NodeReference[string]
	Get         *low.NodeReference[*Operation]
	Put         *low.NodeReference[*Operation]
	Post        *low.NodeReference[*Operation]
	Delete      *low.NodeReference[*Operation]
	Options     *low.NodeReference[*Operation]
	Head        *low.NodeReference[*Operation]
	Patch       *low.NodeReference[*Operation]
	Trace       *low.NodeReference[*Operation]
	Servers     []*low.NodeReference[*Server]
	Parameters  []*low.NodeReference[*Parameter]
	Extensions  map[low.NodeReference[string]]low.NodeReference[any]
}

func (p *Path) Build(root *yaml.Node, idx *index.SpecIndex) error {
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	p.Extensions = extensionMap
	skip := false
	var currentNode *yaml.Node

	var wg sync.WaitGroup
	var errors []error

	var ops []low.NodeReference[*Operation]

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
			p.Get = &opRef
		case PostLabel:
			p.Post = &opRef
		case PutLabel:
			p.Put = &opRef
		case PatchLabel:
			p.Patch = &opRef
		case DeleteLabel:
			p.Delete = &opRef
		case HeadLabel:
			p.Head = &opRef
		case OptionsLabel:
			p.Options = &opRef
		case TraceLabel:
			p.Trace = &opRef
		}
	}

	wg.Wait()

	// all operations have been superficially built,
	// now we need to build out the operation, we will do this asynchronously for speed.
	opBuildChan := make(chan bool)
	opErrorChan := make(chan error)

	var buildOpFunc = func(op low.NodeReference[*Operation], ch chan<- bool, errCh chan<- error) {

		// build out the operation.
		er := op.Value.Build(op.ValueNode, idx)
		if err != nil {
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
			if n == len(ops)-1 {
				break allDone
			}
		}
	}

	return nil
}

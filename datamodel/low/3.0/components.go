package v3

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

const (
	ComponentsLabel = "components"
	SchemasLabel    = "schemas"
)

type Components struct {
	Schemas         low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Schema]]
	Responses       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Response]]
	Parameters      low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Parameter]]
	Examples        low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Example]]
	RequestBodies   low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*RequestBody]]
	Headers         low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]
	SecuritySchemes low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SecurityScheme]]
	Links           low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]]
	Callbacks       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Callback]]
	Extensions      map[low.KeyReference[string]]low.ValueReference[any]
}

func (co *Components) FindSchema(schema string) *low.ValueReference[*Schema] {
	return FindItemInMap[*Schema](schema, co.Schemas.Value)
}

func (co *Components) FindResponse(response string) *low.ValueReference[*Response] {
	return FindItemInMap[*Response](response, co.Responses.Value)
}

func (co *Components) FindParameter(response string) *low.ValueReference[*Parameter] {
	return FindItemInMap[*Parameter](response, co.Parameters.Value)
}

func (co *Components) FindSecurityScheme(sScheme string) *low.ValueReference[*SecurityScheme] {
	return FindItemInMap[*SecurityScheme](sScheme, co.SecuritySchemes.Value)
}

func (co *Components) FindExample(example string) *low.ValueReference[*Example] {
	return FindItemInMap[*Example](example, co.Examples.Value)
}

func (co *Components) FindRequestBody(requestBody string) *low.ValueReference[*RequestBody] {
	return FindItemInMap[*RequestBody](requestBody, co.RequestBodies.Value)
}

func (co *Components) FindHeader(header string) *low.ValueReference[*Header] {
	return FindItemInMap[*Header](header, co.Headers.Value)
}

func (co *Components) FindLink(link string) *low.ValueReference[*Link] {
	return FindItemInMap[*Link](link, co.Links.Value)
}

func (co *Components) FindCallback(callback string) *low.ValueReference[*Callback] {
	return FindItemInMap[*Callback](callback, co.Callbacks.Value)
}

func (co *Components) Build(root *yaml.Node, idx *index.SpecIndex) error {
	co.Extensions = ExtractExtensions(root)

	// build out components asynchronously for speed. There could be some significant weight here.
	skipChan := make(chan bool)
	errorChan := make(chan error)
	paramChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Parameter]])
	schemaChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Schema]])
	responsesChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Response]])
	examplesChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Example]])
	requestBodiesChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*RequestBody]])
	headersChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]])
	securitySchemesChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SecurityScheme]])
	linkChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]])
	callbackChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Callback]])

	go extractComponentValues[*Schema](SchemasLabel, root, skipChan, errorChan, schemaChan, idx)
	go extractComponentValues[*Parameter](ParametersLabel, root, skipChan, errorChan, paramChan, idx)
	go extractComponentValues[*Response](ResponsesLabel, root, skipChan, errorChan, responsesChan, idx)
	go extractComponentValues[*Example](ExamplesLabel, root, skipChan, errorChan, examplesChan, idx)
	go extractComponentValues[*RequestBody](RequestBodiesLabel, root, skipChan, errorChan, requestBodiesChan, idx)
	go extractComponentValues[*Header](HeadersLabel, root, skipChan, errorChan, headersChan, idx)
	go extractComponentValues[*SecurityScheme](SecuritySchemesLabel, root, skipChan, errorChan, securitySchemesChan, idx)
	go extractComponentValues[*Link](LinksLabel, root, skipChan, errorChan, linkChan, idx)
	go extractComponentValues[*Callback](CallbacksLabel, root, skipChan, errorChan, callbackChan, idx)

	n := 0
	total := 9

	var stateCheck = func() bool {
		n++
		if n == total {
			return true
		}
		return false
	}

allDone:
	for {
		select {
		case buildError := <-errorChan:
			return buildError
		case <-skipChan:
			if stateCheck() {
				break allDone
			}
		case params := <-paramChan:
			co.Parameters = params
			if stateCheck() {
				break allDone
			}
		case schemas := <-schemaChan:
			co.Schemas = schemas
			if stateCheck() {
				break allDone
			}
		case responses := <-responsesChan:
			co.Responses = responses
			if stateCheck() {
				break allDone
			}
		case examples := <-examplesChan:
			co.Examples = examples
			if stateCheck() {
				break allDone
			}
		case reqBody := <-requestBodiesChan:
			co.RequestBodies = reqBody
			if stateCheck() {
				break allDone
			}
		case headers := <-headersChan:
			co.Headers = headers
			if stateCheck() {
				break allDone
			}
		case sScheme := <-securitySchemesChan:
			co.SecuritySchemes = sScheme
			if stateCheck() {
				break allDone
			}
		case links := <-linkChan:
			co.Links = links
			if stateCheck() {
				break allDone
			}
		case callbacks := <-callbackChan:
			co.Callbacks = callbacks
			if stateCheck() {
				break allDone
			}
		}
	}

	return nil
}

func extractComponentValues[T low.Buildable[N], N any](label string, root *yaml.Node,
	skip chan bool, errorChan chan<- error, resultChan chan<- low.NodeReference[map[low.KeyReference[string]]low.ValueReference[T]], idx *index.SpecIndex) {
	_, nodeLabel, nodeValue := utils.FindKeyNodeFull(label, root.Content)
	if nodeValue == nil {
		skip <- true
		return
	}
	var currentLabel *yaml.Node
	componentValues := make(map[low.KeyReference[string]]low.ValueReference[T])
	if utils.IsNodeArray(nodeValue) {
		errorChan <- fmt.Errorf("node is array, cannot be used in components: line %d, column %d", nodeValue.Line, nodeValue.Column)
		return
	}
	for i, v := range nodeValue.Content {
		if i%2 == 0 {
			currentLabel = v
			continue
		}
		var n T = new(N)
		_ = BuildModel(v, n)
		err := n.Build(v, idx)
		if err != nil {
			errorChan <- err
		}
		componentValues[low.KeyReference[string]{
			KeyNode: currentLabel,
			Value:   currentLabel.Value,
		}] = low.ValueReference[T]{
			Value:     n,
			ValueNode: v,
		}
	}
	results := low.NodeReference[map[low.KeyReference[string]]low.ValueReference[T]]{
		KeyNode:   nodeLabel,
		ValueNode: nodeValue,
		Value:     componentValues,
	}
	resultChan <- results
}

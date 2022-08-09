package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
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

func (co *Components) Build(root *yaml.Node) error {
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	co.Extensions = extensionMap

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

	go extractComponentValues[*Schema](SchemasLabel, root, skipChan, errorChan, schemaChan)
	go extractComponentValues[*Parameter](ParametersLabel, root, skipChan, errorChan, paramChan)
	go extractComponentValues[*Response](ResponsesLabel, root, skipChan, errorChan, responsesChan)
	go extractComponentValues[*Example](ExamplesLabel, root, skipChan, errorChan, examplesChan)
	go extractComponentValues[*RequestBody](RequestBodiesLabel, root, skipChan, errorChan, requestBodiesChan)
	go extractComponentValues[*Header](HeadersLabel, root, skipChan, errorChan, headersChan)
	go extractComponentValues[*SecurityScheme](SecuritySchemesLabel, root, skipChan, errorChan, securitySchemesChan)
	go extractComponentValues[*Link](LinksLabel, root, skipChan, errorChan, linkChan)
	go extractComponentValues[*Callback](CallbacksLabel, root, skipChan, errorChan, callbackChan)

	n := 0
	total := 9
allDone:
	for {
		select {
		case buildError := <-errorChan:
			return buildError
		case <-skipChan:
			n++
			if n == total {
				break allDone
			}
		case params := <-paramChan:
			co.Parameters = params
			n++
			if n == total {
				break allDone
			}
		case schemas := <-schemaChan:
			co.Schemas = schemas
			n++
			if n == total {
				break allDone
			}
		case responses := <-responsesChan:
			co.Responses = responses
			n++
			if n == total {
				break allDone
			}
		case examples := <-examplesChan:
			co.Examples = examples
			n++
			if n == total {
				break allDone
			}
		case reqBody := <-requestBodiesChan:
			co.RequestBodies = reqBody
			n++
			if n == total {
				break allDone
			}
		case headers := <-headersChan:
			co.Headers = headers
			n++
			if n == total {
				break allDone
			}
		case sScheme := <-securitySchemesChan:
			co.SecuritySchemes = sScheme
			n++
			if n == total {
				break allDone
			}
		case links := <-linkChan:
			co.Links = links
			n++
			if n == total {
				break allDone
			}
		case callbacks := <-callbackChan:
			co.Callbacks = callbacks
			n++
			if n == total {
				break allDone
			}
		}
	}

	return nil
}

func extractComponentValues[T low.Buildable[N], N any](label string, root *yaml.Node,
	skip chan bool, errorChan chan<- error, resultChan chan<- low.NodeReference[map[low.KeyReference[string]]low.ValueReference[T]]) {
	_, nodeLabel, nodeValue := utils.FindKeyNodeFull(label, root.Content)
	if nodeValue == nil {
		skip <- true
		return
	}
	var currentLabel *yaml.Node
	componentValues := make(map[low.KeyReference[string]]low.ValueReference[T])
	for i, v := range nodeValue.Content {
		if i%2 == 0 {
			currentLabel = v
			continue
		}
		var n T = new(N)
		err := BuildModel(v, n)
		if err != nil {
			errorChan <- err
		}
		err = n.Build(v)
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

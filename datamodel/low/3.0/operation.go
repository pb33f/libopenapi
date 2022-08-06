package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
)

const (
	ParametersLabel = "parameters"
)

type Operation struct {
	Tags         []low.NodeReference[string]
	Summary      low.NodeReference[string]
	Description  low.NodeReference[string]
	ExternalDocs low.NodeReference[*ExternalDoc]
	OperationId  low.NodeReference[string]
	Parameters   []low.NodeReference[*Parameter]
	RequestBody  low.NodeReference[*RequestBody]
	Responses    low.NodeReference[*Responses]
	Callbacks    map[low.KeyReference[string]]low.ValueReference[*Callback]
	Deprecated   low.NodeReference[bool]
	Security     []low.NodeReference[*SecurityRequirement]
	Servers      []low.NodeReference[*Server]
	Extensions   map[low.KeyReference[string]]low.ValueReference[any]
}

func (o *Operation) Build(root *yaml.Node) error {

	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	o.Extensions = extensionMap

	// extract externalDocs
	extDocs, dErr := ExtractObject[ExternalDoc](ExternalDocsLabel, root)
	if dErr != nil {
		return dErr
	}
	o.ExternalDocs = extDocs

	// extract parameters
	params, pErr := ExtractArray[*Parameter](ParametersLabel, root)
	if pErr != nil {
		return pErr
	}
	if params != nil {
		o.Parameters = params
	}

	return nil
}

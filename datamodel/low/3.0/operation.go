package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

const (
	ParametersLabel    = "parameters"
	RequestBodyLabel   = "requestBody"
	RequestBodiesLabel = "requestBodies"
	ResponsesLabel     = "responses"
	CallbacksLabel     = "callbacks"
)

type Operation struct {
	Tags         low.NodeReference[low.NodeReference[string]]
	Summary      low.NodeReference[string]
	Description  low.NodeReference[string]
	ExternalDocs low.NodeReference[*ExternalDoc]
	OperationId  low.NodeReference[string]
	Parameters   low.NodeReference[[]low.ValueReference[*Parameter]]
	RequestBody  low.NodeReference[*RequestBody]
	Responses    low.NodeReference[*Responses]
	Callbacks    low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Callback]]
	Deprecated   low.NodeReference[bool]
	Security     low.NodeReference[*SecurityRequirement]
	Servers      low.NodeReference[[]low.ValueReference[*Server]]
	Extensions   map[low.KeyReference[string]]low.ValueReference[any]
}

func (o *Operation) Build(root *yaml.Node, idx *index.SpecIndex) error {
	o.Extensions = ExtractExtensions(root)

	// extract externalDocs
	extDocs, dErr := ExtractObject[*ExternalDoc](ExternalDocsLabel, root, idx)
	if dErr != nil {
		return dErr
	}
	o.ExternalDocs = extDocs

	// extract parameters
	params, ln, vn, pErr := ExtractArray[*Parameter](ParametersLabel, root, idx)
	if pErr != nil {
		return pErr
	}
	if params != nil {
		o.Parameters = low.NodeReference[[]low.ValueReference[*Parameter]]{
			Value:     params,
			KeyNode:   ln,
			ValueNode: vn,
		}
	}

	// extract request body
	rBody, rErr := ExtractObject[*RequestBody](RequestBodyLabel, root, idx)
	if rErr != nil {
		return rErr
	}
	o.RequestBody = rBody

	// extract responses
	respBody, respErr := ExtractObject[*Responses](ResponsesLabel, root, idx)
	if respErr != nil {
		return rErr
	}
	o.Responses = respBody

	// extract callbacks
	callbacks, cbL, cbN, cbErr := ExtractMapFlat[*Callback](CallbacksLabel, root, idx)
	if cbErr != nil {
		return cbErr
	}
	if callbacks != nil {
		o.Callbacks = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Callback]]{
			Value:     callbacks,
			KeyNode:   cbL,
			ValueNode: cbN,
		}
	}

	// extract security
	sec, sErr := ExtractObject[*SecurityRequirement](SecurityLabel, root, idx)
	if sErr != nil {
		return sErr
	}
	o.Security = sec

	// extract servers
	servers, sl, sn, serErr := ExtractArray[*Server](ServersLabel, root, idx)
	if serErr != nil {
		return serErr
	}
	if servers != nil {
		o.Servers = low.NodeReference[[]low.ValueReference[*Server]]{
			Value:     servers,
			KeyNode:   sl,
			ValueNode: sn,
		}
	}
	return nil
}

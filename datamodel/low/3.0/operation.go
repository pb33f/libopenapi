package v3

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

const (
	ParametersLabel = "parameters"
)

type Operation struct {
	Tags         []low.NodeReference[string]
	Summary      low.NodeReference[string]
	Description  low.NodeReference[string]
	ExternalDocs *low.NodeReference[*ExternalDoc]
	OperationId  low.NodeReference[string]
	Parameters   []low.NodeReference[*Parameter]
	RequestBody  *low.NodeReference[*RequestBody]
	Responses    *low.NodeReference[*Responses]
	Callbacks    map[low.NodeReference[string]]low.NodeReference[*Callback]
	Deprecated   *low.NodeReference[bool]
	Security     []low.NodeReference[*SecurityRequirement]
	Servers      []low.NodeReference[*Server]
	Extensions   map[low.NodeReference[string]]low.NodeReference[any]
}

func (o *Operation) Build(root *yaml.Node, idx *index.SpecIndex) error {

	extensionMap, err := datamodel.ExtractExtensions(root)
	if err != nil {
		return err
	}
	o.Extensions = extensionMap

	// extract external docs
	_, ln, exDocs := utils.FindKeyNodeFull(ExternalDocsLabel, root.Content)
	if exDocs != nil {
		var externalDoc ExternalDoc
		err = datamodel.BuildModel(exDocs, &externalDoc)
		if err != nil {
			return err
		}
		o.ExternalDocs = &low.NodeReference[*ExternalDoc]{
			Value:     &externalDoc,
			KeyNode:   ln,
			ValueNode: exDocs,
		}
	}

	// build parameters
	_, paramLabel, paramNode := utils.FindKeyNodeFull(ParametersLabel, root.Content)
	if paramNode != nil && paramLabel != nil {
		var params []low.NodeReference[*Parameter]

		for _, pN := range paramNode.Content {
			var param Parameter
			err = datamodel.BuildModel(pN, &param)
			if err != nil {
				return err
			}
			err = param.Build(pN, idx)
			if err != nil {
				return err
			}
			params = append(params, low.NodeReference[*Parameter]{
				Value:     &param,
				ValueNode: paramNode,
				KeyNode:   paramLabel,
			})
		}
		o.Parameters = params
	}

	return nil
}

package v3

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

const (
	VariablesLabel = "variables"
	ServersLabel   = "servers"
)

type Server struct {
	URL         low.NodeReference[string]
	Description low.NodeReference[string]
	Variables   low.NodeReference[map[string]low.NodeReference[*ServerVariable]]
}

func (s *Server) Build(root *yaml.Node) error {
	kn, vars := utils.FindKeyNode(VariablesLabel, root.Content)
	if vars == nil {
		return nil
	}
	variablesMap := make(map[string]low.NodeReference[*ServerVariable])
	if utils.IsNodeMap(vars) {
		var currentNode string
		var keyNode *yaml.Node
		for i, varNode := range vars.Content {
			if i%2 == 0 {
				currentNode = varNode.Value
				keyNode = varNode
				continue
			}
			variable := ServerVariable{}
			err := datamodel.BuildModel(varNode, &variable)
			if err != nil {
				return err
			}
			variablesMap[currentNode] = low.NodeReference[*ServerVariable]{
				ValueNode: varNode,
				KeyNode:   keyNode,
				Value:     &variable,
			}
		}
		s.Variables = low.NodeReference[map[string]low.NodeReference[*ServerVariable]]{
			KeyNode:   kn,
			ValueNode: vars,
			Value:     variablesMap,
		}
	}
	return nil
}

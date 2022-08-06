package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
)

type SecurityScheme struct {
	Type             low.NodeReference[string]
	Description      low.NodeReference[string]
	Name             low.NodeReference[string]
	In               low.NodeReference[string]
	Scheme           low.NodeReference[string]
	BearerFormat     low.NodeReference[string]
	Flows            low.NodeReference[*OAuthFlows]
	OpenIdConnectURL low.NodeReference[string]
	Extensions       map[low.KeyReference[string]]low.ValueReference[any]
}

func (ss *SecurityScheme) Build(root *yaml.Node) error {
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	ss.Extensions = extensionMap
	return nil
}

type SecurityRequirement struct {
	Value low.NodeReference[[]low.ValueReference[string]]
}

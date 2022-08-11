package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

const (
	ImplicitLabel          = "implicit"
	PasswordLabel          = "password"
	ClientCredentialsLabel = "clientCredentials"
	AuthorizationCodeLabel = "authorizationCode"
	ScopesLabel            = "scopes"
)

type OAuthFlows struct {
	Implicit          low.NodeReference[*OAuthFlow]
	Password          low.NodeReference[*OAuthFlow]
	ClientCredentials low.NodeReference[*OAuthFlow]
	AuthorizationCode low.NodeReference[*OAuthFlow]
	Extensions        map[low.KeyReference[string]]low.ValueReference[any]
}

func (o *OAuthFlows) Build(root *yaml.Node, idx *index.SpecIndex) error {
	o.Extensions = ExtractExtensions(root)

	v, vErr := ExtractObject[*OAuthFlow](ImplicitLabel, root, idx)
	if vErr != nil {
		return vErr
	}
	o.Implicit = v

	v, vErr = ExtractObject[*OAuthFlow](PasswordLabel, root, idx)
	if vErr != nil {
		return vErr
	}
	o.Password = v

	v, vErr = ExtractObject[*OAuthFlow](ClientCredentialsLabel, root, idx)
	if vErr != nil {
		return vErr
	}
	o.ClientCredentials = v

	v, vErr = ExtractObject[*OAuthFlow](AuthorizationCodeLabel, root, idx)
	if vErr != nil {
		return vErr
	}
	o.AuthorizationCode = v
	return nil

}

type OAuthFlow struct {
	AuthorizationUrl low.NodeReference[string]
	TokenURL         low.NodeReference[string]
	RefreshURL       low.NodeReference[string]
	Scopes           low.NodeReference[map[low.KeyReference[string]]low.ValueReference[string]]
	Extensions       map[low.KeyReference[string]]low.ValueReference[any]
}

func (o *OAuthFlow) FindScope(scope string) *low.ValueReference[string] {
	return FindItemInMap[string](scope, o.Scopes.Value)
}

func (o *OAuthFlow) Build(root *yaml.Node, idx *index.SpecIndex) error {
	o.Extensions = ExtractExtensions(root)

	var currSec *yaml.Node

	// extract scopes
	_, scopeLabel, scopeNode := utils.FindKeyNodeFull(ScopesLabel, root.Content)
	if scopeNode != nil {
		res := make(map[low.KeyReference[string]]low.ValueReference[string])
		for i, r := range scopeNode.Content {
			if i%2 == 0 {
				currSec = r
				continue
			}
			res[low.KeyReference[string]{
				Value:   currSec.Value,
				KeyNode: currSec,
			}] = low.ValueReference[string]{
				Value:     r.Value,
				ValueNode: r,
			}
		}
		if len(res) > 0 {
			o.Scopes = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[string]]{
				Value:     res,
				KeyNode:   scopeLabel,
				ValueNode: scopeNode,
			}
		}
	}

	return nil
}

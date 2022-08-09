package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
)

const (
	ImplicitLabel          = "implicit"
	PasswordLabel          = "password"
	ClientCredentialsLabel = "clientCredentials"
	AuthorizationCodeLabel = "authorizationCode"
)

type OAuthFlows struct {
	Implicit          low.NodeReference[*OAuthFlow]
	Password          low.NodeReference[*OAuthFlow]
	ClientCredentials low.NodeReference[*OAuthFlow]
	AuthorizationCode low.NodeReference[*OAuthFlow]
	Extensions        map[low.KeyReference[string]]low.ValueReference[any]
}

func (o *OAuthFlows) Build(root *yaml.Node) error {
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	o.Extensions = extensionMap

	v, vErr := ExtractObject[*OAuthFlow](ImplicitLabel, root)
	if vErr != nil {
		return vErr
	}
	o.Implicit = v

	v, vErr = ExtractObject[*OAuthFlow](PasswordLabel, root)
	if vErr != nil {
		return vErr
	}
	o.Password = v

	v, vErr = ExtractObject[*OAuthFlow](ClientCredentialsLabel, root)
	if vErr != nil {
		return vErr
	}
	o.ClientCredentials = v

	v, vErr = ExtractObject[*OAuthFlow](AuthorizationCodeLabel, root)
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
	Scopes           map[low.KeyReference[string]]low.ValueReference[string]
	Extensions       map[low.KeyReference[string]]low.ValueReference[any]
}

func (o *OAuthFlow) Build(root *yaml.Node) error {
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	o.Extensions = extensionMap
	return nil
}

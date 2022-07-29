package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type OAuthFlows struct {
    Node              *yaml.Node
    Implicit          OAuthFlow
    Password          OAuthFlow
    ClientCredentials OAuthFlow
    AuthorizationCode OAuthFlow
    Extensions        map[string]low.ObjectReference
}

type OAuthFlow struct {
    Node             *yaml.Node
    AuthorizationUrl low.NodeReference[string]
    TokenURL         low.NodeReference[string]
    RefreshURL       low.NodeReference[string]
    Scopes           map[string]string
    Extensions       map[string]low.ObjectReference
}

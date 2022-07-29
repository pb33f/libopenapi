package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type SecurityScheme struct {
    Node             *yaml.Node
    Type             low.NodeReference[string]
    Description      low.NodeReference[string]
    Name             low.NodeReference[string]
    In               low.NodeReference[string]
    Scheme           low.NodeReference[string]
    BearerFormat     low.NodeReference[string]
    Flows            OAuthFlows
    OpenIdConnectURL low.NodeReference[string]
    Extensions       map[string]low.ObjectReference
}

type SecurityRequirement struct {
    Node  *yaml.Node
    Value []low.NodeReference[string]
}

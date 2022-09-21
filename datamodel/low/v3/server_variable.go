package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
)

// ServerVariable represents a low-level OpenAPI 3+ ServerVariable object.
//
// ServerVariable is an object representing a Server Variable for server URL template substitution.
// - https://spec.openapis.org/oas/v3.1.0#server-variable-object
//
// This is the only struct that is not Buildable, it's not used by anything other than a Server instance,
// and it has nothing to build that requires it to be buildable.
type ServerVariable struct {
	Enum        []low.NodeReference[string]
	Default     low.NodeReference[string]
	Description low.NodeReference[string]
}

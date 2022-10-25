package v3

import (
	"crypto/sha256"
	"github.com/pb33f/libopenapi/datamodel/low"
	"strings"
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

// Hash will return a consistent SHA256 Hash of the ServerVariable object
func (s *ServerVariable) Hash() [32]byte {
	var f []string
	for k := range s.Enum {
		f = append(f, s.Enum[k].Value)
	}
	if !s.Default.IsEmpty() {
		f = append(f, s.Default.Value)
	}
	if !s.Description.IsEmpty() {
		f = append(f, s.Description.Value)
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"crypto/sha256"

	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// SchemaProxy exists as a stub that will create a Schema once (and only once) the Schema() method is called.
//
// Why use a Proxy design?
//
// There are three reasons.
//
// 1. Circular References and Endless Loops.
//
// JSON Schema allows for references to be used. This means references can loop around and create infinite recursive
// structures, These 'Circular references' technically mean a schema can NEVER be resolved, not without breaking the
// loop somewhere along the chain.
//
// Polymorphism in the form of 'oneOf' and 'anyOf' in version 3+ only exacerbates the problem.
//
// These circular traps can be discovered using the resolver, however it's still not enough to stop endless loops and
// endless goroutine spawning. A proxy design means that resolving occurs on demand and runs down a single level only.
// preventing any run-away loops.
//
// 2. Performance
//
// Even without circular references, Polymorphism creates large additional resolving chains that take a long time
// and slow things down when building. By preventing recursion through every polymorphic item, building models is kept
// fast and snappy, which is desired for realtime processing of specs.
//
//   - Q: Yeah, but, why not just use state to avoiding re-visiting seen polymorphic nodes?
//   - A: It's slow, takes up memory and still has runaway potential in very, very long chains.
//
// 3. Short Circuit Errors.
//
// Schemas are where things can get messy, mainly because the Schema standard changes between versions, and
// it's not actually JSONSchema until 3.1, so lots of times a bad schema will break parsing. Errors are only found
// when a schema is needed, so the rest of the document is parsed and ready to use.
type SchemaProxy struct {
	kn              *yaml.Node
	vn              *yaml.Node
	idx             *index.SpecIndex
	rendered        *Schema
	buildError      error
	isReference     bool   // Is the schema underneath originally a $ref?
	referenceLookup string // If the schema is a $ref, what's its name?
}

// Build will prepare the SchemaProxy for rendering, it does not build the Schema, only sets up internal state.
// Key maybe nil if absent.
func (sp *SchemaProxy) Build(key, value *yaml.Node, idx *index.SpecIndex) error {
	sp.kn = key
	sp.vn = value
	sp.idx = idx
	if rf, _, r := utils.IsNodeRefValue(value); rf {
		sp.isReference = true
		sp.referenceLookup = r
	}
	return nil
}

// Schema will first check if this SchemaProxy has already rendered the schema, and return the pre-rendered version
// first.
//
// If this is the first run of Schema(), then the SchemaProxy will create a new Schema from the underlying
// yaml.Node. Once built out, the SchemaProxy will record that Schema as rendered and store it for later use,
// (this is what is we mean when we say 'pre-rendered').
//
// Schema() then returns the newly created Schema.
//
// If anything goes wrong during the build, then nothing is returned and the error that occurred can
// be retrieved by using GetBuildError()
func (sp *SchemaProxy) Schema() *Schema {
	if sp.rendered != nil {
		return sp.rendered
	}
	schema := new(Schema)
	utils.CheckForMergeNodes(sp.vn)
	err := schema.Build(sp.vn, sp.idx)
	if err != nil {
		sp.buildError = err
		return nil
	}
	schema.ParentProxy = sp // https://github.com/pb33f/libopenapi/issues/29
	sp.rendered = schema
	return schema
}

// GetBuildError returns the build error that was set when Schema() was called. If Schema() has not been run, or
// there were no errors during build, then nil will be returned.
func (sp *SchemaProxy) GetBuildError() error {
	return sp.buildError
}

// IsSchemaReference returns true if the Schema that this SchemaProxy represents, is actually a reference to
// a Schema contained within Components or Definitions. There is no difference in the mechanism used to resolve the
// Schema when calling Schema(), however if we want to know if this schema was originally a reference, we won't
// be able to determine that from the model, without this bit.
func (sp *SchemaProxy) IsSchemaReference() bool {
	return sp.isReference
}

// IsReference is an alias for IsSchemaReference() except it's compatible wih the IsReferenced interface type.
func (sp *SchemaProxy) IsReference() bool {
	return sp.IsSchemaReference()
}

// GetReference is an alias for GetSchemaReference() except it's compatible wih the IsReferenced interface type.
func (sp *SchemaProxy) GetReference() string {
	return sp.GetSchemaReference()
}

// SetReference will set the reference lookup for this SchemaProxy.
func (sp *SchemaProxy) SetReference(ref string) {
	sp.referenceLookup = ref
}

// GetSchemaReference will return the lookup defined by the $ref that this schema points to. If the schema
// is inline, and not a reference, then this method returns an empty string. Only useful when combined with
// IsSchemaReference()
func (sp *SchemaProxy) GetSchemaReference() string {
	return sp.referenceLookup
}

// GetKeyNode will return the yaml.Node pointer that is a key for value node.
func (sp *SchemaProxy) GetKeyNode() *yaml.Node {
	return sp.kn
}

// GetValueNode will return the yaml.Node pointer used by the proxy to generate the Schema.
func (sp *SchemaProxy) GetValueNode() *yaml.Node {
	return sp.vn
}

// Hash will return a consistent SHA256 Hash of the SchemaProxy object (it will resolve it)
func (sp *SchemaProxy) Hash() [32]byte {
	if sp.rendered != nil {
		if !sp.isReference {
			return sp.rendered.Hash()
		}
	} else {
		if !sp.isReference {
			// only resolve this proxy if it's not a ref.
			sch := sp.Schema()
			sp.rendered = sch
			return sch.Hash()
		}
	}
	// hash reference value only, do not resolve!
	return sha256.Sum256([]byte(sp.referenceLookup))
}

// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"

	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

// FindItemInMap accepts a string key and a collection of KeyReference[string] and ValueReference[T]. Every
// KeyReference will have its value checked against the string key and if there is a match, it will be returned.
func FindItemInMap[T any](item string, collection map[KeyReference[string]]ValueReference[T]) *ValueReference[T] {
	for n, o := range collection {
		if n.Value == item {
			return &o
		}
		if strings.ToLower(n.Value) == strings.ToLower(item) {
			return &o
		}
	}
	return nil
}

// helper function to generate a list of all the things an index should be searched for.
func generateIndexCollection(idx *index.SpecIndex) []func() map[string]*index.Reference {
	return []func() map[string]*index.Reference{
		idx.GetAllComponentSchemas,
		idx.GetMappedReferences,
		idx.GetAllExternalDocuments,
		idx.GetAllParameters,
		idx.GetAllHeaders,
		idx.GetAllCallbacks,
		idx.GetAllLinks,
		idx.GetAllExamples,
		idx.GetAllRequestBodies,
		idx.GetAllResponses,
		idx.GetAllSecuritySchemes,
	}
}

// LocateRefNode will perform a complete lookup for a $ref node. This function searches the entire index for
// the reference being supplied. If there is a match found, the reference *yaml.Node is returned.
func LocateRefNode(root *yaml.Node, idx *index.SpecIndex) (*yaml.Node, error) {
	if rf, _, rv := utils.IsNodeRefValue(root); rf {

		// run through everything and return as soon as we find a match.
		// this operates as fast as possible as ever
		collections := generateIndexCollection(idx)

		// if there are any external indexes being used by remote
		// documents, then we need to search through them also.
		externalIndexes := idx.GetAllExternalIndexes()
		if len(externalIndexes) > 0 {
			var extCollection []func() map[string]*index.Reference
			for _, extIndex := range externalIndexes {
				extCollection = generateIndexCollection(extIndex)
				collections = append(collections, extCollection...)
			}
		}

		var found map[string]*index.Reference
		for _, collection := range collections {
			found = collection()
			if found != nil && found[rv] != nil {

				// if this is a ref node, we need to keep diving
				// until we hit something that isn't a ref.
				if jh, _, _ := utils.IsNodeRefValue(found[rv].Node); jh {
					// if this node is circular, stop drop and roll.
					if !IsCircular(found[rv].Node, idx) {
						return LocateRefNode(found[rv].Node, idx)
					} else {
						return found[rv].Node, fmt.Errorf("circular reference '%s' found during lookup at line "+
							"%d, column %d, It cannot be resolved",
							GetCircularReferenceResult(found[rv].Node, idx).GenerateJourneyPath(),
							found[rv].Node.Line,
							found[rv].Node.Column)
					}
				}
				return found[rv].Node, nil
			}
		}

		// perform a search for the reference in the index
		foundRefs := idx.SearchIndexForReference(rv)
		if len(foundRefs) > 0 {
			return foundRefs[0].Node, nil
		}

		// let's try something else to find our references.

		// cant be found? last resort is to try a path lookup
		_, friendly := utils.ConvertComponentIdIntoFriendlyPathSearch(rv)
		if friendly != "" {
			path, err := yamlpath.NewPath(friendly)
			if err == nil {
				nodes, fErr := path.Find(idx.GetRootNode())
				if fErr == nil {
					if len(nodes) > 0 {
						return nodes[0], nil
					}
				}
			}
		}
		return nil, fmt.Errorf("reference '%s' at line %d, column %d was not found",
			rv, root.Line, root.Column)
	}
	return nil, nil
}

// ExtractObjectRaw will extract a typed Buildable[N] object from a root yaml.Node. The 'raw' aspect is
// that there is no NodeReference wrapper around the result returned, just the raw object.
func ExtractObjectRaw[T Buildable[N], N any](root *yaml.Node, idx *index.SpecIndex) (T, error, bool, string) {
	var circError error
	var isReference bool
	var referenceValue string
	if h, _, rv := utils.IsNodeRefValue(root); h {
		ref, err := LocateRefNode(root, idx)
		if ref != nil {
			root = ref
			isReference = true
			referenceValue = rv
			if err != nil {
				circError = err
			}
		} else {
			if err != nil {
				return nil, fmt.Errorf("object extraction failed: %s", err.Error()), isReference, referenceValue
			}
		}
	}
	var n T = new(N)
	err := BuildModel(root, n)
	if err != nil {
		return n, err, isReference, referenceValue
	}
	err = n.Build(root, idx)
	if err != nil {
		return n, err, isReference, referenceValue
	}

	// if this is a reference, keep track of the reference in the value
	if isReference {
		SetReference(n, referenceValue)
	}

	// do we want to throw an error as well if circular error reporting is on?
	if circError != nil && !idx.AllowCircularReferenceResolving() {
		return n, circError, isReference, referenceValue
	}
	return n, nil, isReference, referenceValue
}

// ExtractObject will extract a typed Buildable[N] object from a root yaml.Node. The result is wrapped in a
// NodeReference[T] that contains the key node found and value node found when looking up the reference.
func ExtractObject[T Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) (NodeReference[T], error) {
	var ln, vn *yaml.Node
	var circError error
	var isReference bool
	var referenceValue string
	if rf, rl, refVal := utils.IsNodeRefValue(root); rf {
		ref, err := LocateRefNode(root, idx)
		if ref != nil {
			vn = ref
			ln = rl
			isReference = true
			referenceValue = refVal
			if err != nil {
				circError = err
			}
		} else {
			if err != nil {
				return NodeReference[T]{}, fmt.Errorf("object extraction failed: %s", err.Error())
			}
		}
	} else {
		_, ln, vn = utils.FindKeyNodeFull(label, root.Content)
		if vn != nil {
			if h, _, rVal := utils.IsNodeRefValue(vn); h {
				ref, lerr := LocateRefNode(vn, idx)
				if ref != nil {
					vn = ref
					isReference = true
					referenceValue = rVal
					if lerr != nil {
						circError = lerr
					}
				} else {
					if lerr != nil {
						return NodeReference[T]{}, fmt.Errorf("object extraction failed: %s", lerr.Error())
					}
				}
			}
		}
	}
	var n T = new(N)
	err := BuildModel(vn, n)
	if err != nil {
		return NodeReference[T]{}, err
	}
	if ln == nil {
		return NodeReference[T]{}, nil
	}
	err = n.Build(vn, idx)
	if err != nil {
		return NodeReference[T]{}, err
	}

	// if this is a reference, keep track of the reference in the value
	if isReference {
		SetReference(n, referenceValue)
	}

	res := NodeReference[T]{
		Value:         n,
		KeyNode:       ln,
		ValueNode:     vn,
		ReferenceNode: isReference,
		Reference:     referenceValue,
	}

	// do we want to throw an error as well if circular error reporting is on?
	if circError != nil && !idx.AllowCircularReferenceResolving() {
		return res, circError
	}
	return res, nil
}

func SetReference(obj any, ref string) {
	if obj == nil {
		return
	}
	if r, ok := obj.(IsReferenced); ok {
		r.SetReference(ref)
	}
}

// ExtractArray will extract a slice of []ValueReference[T] from a root yaml.Node that is defined as a sequence.
// Used when the value being extracted is an array.
func ExtractArray[T Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) ([]ValueReference[T],
	*yaml.Node, *yaml.Node, error,
) {
	var ln, vn *yaml.Node
	var circError error
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		ref, err := LocateRefNode(root, idx)
		if ref != nil {
			vn = ref
			ln = rl
			if err != nil {
				circError = err
			}
		} else {
			return []ValueReference[T]{}, nil, nil, fmt.Errorf("array build failed: reference cannot be found: %s",
				root.Content[1].Value)
		}
	} else {
		_, ln, vn = utils.FindKeyNodeFullTop(label, root.Content)
		if vn != nil {
			if h, _, _ := utils.IsNodeRefValue(vn); h {
				ref, err := LocateRefNode(vn, idx)
				if ref != nil {
					vn = ref
					//referenceValue = rVal
					if err != nil {
						circError = err
					}
				} else {
					if err != nil {
						return []ValueReference[T]{}, nil, nil, fmt.Errorf("array build failed: reference cannot be found: %s",
							err.Error())
					}
				}
			}
		}
	}

	var items []ValueReference[T]
	if vn != nil && ln != nil {
		if !utils.IsNodeArray(vn) {
			return []ValueReference[T]{}, nil, nil, fmt.Errorf("array build failed, input is not an array, line %d, column %d", vn.Line, vn.Column)
		}
		for _, node := range vn.Content {
			localReferenceValue := ""
			//localIsReference := false

			if rf, _, rv := utils.IsNodeRefValue(node); rf {
				refg, err := LocateRefNode(node, idx)
				if refg != nil {
					node = refg
					//localIsReference = true
					localReferenceValue = rv
					if err != nil {
						circError = err
					}
				} else {
					if err != nil {
						return []ValueReference[T]{}, nil, nil, fmt.Errorf("array build failed: reference cannot be found: %s",
							err.Error())
					}
				}
			}
			var n T = new(N)
			err := BuildModel(node, n)
			if err != nil {
				return []ValueReference[T]{}, ln, vn, err
			}
			berr := n.Build(node, idx)
			if berr != nil {
				return nil, ln, vn, berr
			}

			if localReferenceValue != "" {
				SetReference(n, localReferenceValue)
			}

			items = append(items, ValueReference[T]{
				Value:         n,
				ValueNode:     node,
				ReferenceNode: localReferenceValue != "",
				Reference:     localReferenceValue,
			})
		}
	}
	// include circular errors?
	if circError != nil && !idx.AllowCircularReferenceResolving() {
		return items, ln, vn, circError
	}
	return items, ln, vn, nil
}

// ExtractExample will extract a value supplied as an example into a NodeReference. Value can be anything.
// the node value is untyped, so casting will be required when trying to use it.
func ExtractExample(expNode, expLabel *yaml.Node) NodeReference[any] {
	ref := NodeReference[any]{Value: expNode.Value, KeyNode: expLabel, ValueNode: expNode}
	if utils.IsNodeMap(expNode) {
		var decoded map[string]interface{}
		_ = expNode.Decode(&decoded)
		ref.Value = decoded
	}
	if utils.IsNodeArray(expNode) {
		var decoded []interface{}
		_ = expNode.Decode(&decoded)
		ref.Value = decoded
	}
	return ref
}

// ExtractMapNoLookupExtensions will extract a map of KeyReference and ValueReference from a root yaml.Node. The 'NoLookup' part
// refers to the fact that there is no key supplied as part of the extraction, there  is no lookup performed and the
// root yaml.Node pointer is used directly. Pass a true bit to includeExtensions to include extension keys in the map.
//
// This is useful when the node to be extracted, is already known and does not require a search.
func ExtractMapNoLookupExtensions[PT Buildable[N], N any](
	root *yaml.Node,
	idx *index.SpecIndex,
	includeExtensions bool,
) (map[KeyReference[string]]ValueReference[PT], error) {
	valueMap := make(map[KeyReference[string]]ValueReference[PT])
	var circError error
	if utils.IsNodeMap(root) {
		var currentKey *yaml.Node
		skip := false
		for i, node := range root.Content {
			if !includeExtensions {
				if strings.HasPrefix(strings.ToLower(node.Value), "x-") {
					skip = true
					continue
				}
			}
			if skip {
				skip = false
				continue
			}
			if i%2 == 0 {
				currentKey = node
				continue
			}

			var isReference bool
			var referenceValue string
			// if value is a reference, we have to look it up in the index!
			if h, _, rv := utils.IsNodeRefValue(node); h {
				ref, err := LocateRefNode(node, idx)
				if ref != nil {
					node = ref
					isReference = true
					referenceValue = rv
					if err != nil {
						circError = err
					}
				} else {
					if err != nil {
						return nil, fmt.Errorf("map build failed: reference cannot be found: %s", err.Error())
					}
				}
			}

			var n PT = new(N)
			err := BuildModel(node, n)
			if err != nil {
				return nil, err
			}
			berr := n.Build(node, idx)
			if berr != nil {
				return nil, berr
			}
			if isReference {
				SetReference(n, referenceValue)
			}
			if currentKey != nil {
				valueMap[KeyReference[string]{
					Value:   currentKey.Value,
					KeyNode: currentKey,
				}] = ValueReference[PT]{
					Value:     n,
					ValueNode: node,
					//IsReference: isReference,
					Reference: referenceValue,
				}
			}
		}
	}
	if circError != nil && !idx.AllowCircularReferenceResolving() {
		return valueMap, circError
	}
	return valueMap, nil

}

// ExtractMapNoLookup will extract a map of KeyReference and ValueReference from a root yaml.Node. The 'NoLookup' part
// refers to the fact that there is no key supplied as part of the extraction, there  is no lookup performed and the
// root yaml.Node pointer is used directly.
//
// This is useful when the node to be extracted, is already known and does not require a search.
func ExtractMapNoLookup[PT Buildable[N], N any](
	root *yaml.Node,
	idx *index.SpecIndex,
) (map[KeyReference[string]]ValueReference[PT], error) {
	return ExtractMapNoLookupExtensions[PT, N](root, idx, false)
}

type mappingResult[T any] struct {
	k KeyReference[string]
	v ValueReference[T]
}

// ExtractMapExtensions will extract a map of KeyReference and ValueReference from a root yaml.Node. The 'label' is
// used to locate the node to be extracted from the root node supplied. Supply a bit to decide if extensions should
// be included or not. required in some use cases.
//
// The second return value is the yaml.Node found for the 'label' and the third return value is the yaml.Node
// found for the value extracted from the label node.
func ExtractMapExtensions[PT Buildable[N], N any](
	label string,
	root *yaml.Node,
	idx *index.SpecIndex,
	extensions bool,
) (map[KeyReference[string]]ValueReference[PT], *yaml.Node, *yaml.Node, error) {
	//var isReference bool
	var referenceValue string
	var labelNode, valueNode *yaml.Node
	var circError error
	if rf, rl, rv := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref, err := LocateRefNode(root, idx)
		if ref != nil {
			valueNode = ref
			labelNode = rl
			//isReference = true
			referenceValue = rv
			if err != nil {
				circError = err
			}
		} else {
			return nil, labelNode, valueNode, fmt.Errorf("map build failed: reference cannot be found: %s",
				root.Content[1].Value)
		}
	} else {
		_, labelNode, valueNode = utils.FindKeyNodeFull(label, root.Content)
		if valueNode != nil {
			if h, _, rvt := utils.IsNodeRefValue(valueNode); h {
				ref, err := LocateRefNode(valueNode, idx)
				if ref != nil {
					valueNode = ref
					//isReference = true
					referenceValue = rvt
					if err != nil {
						circError = err
					}
				} else {
					if err != nil {
						return nil, labelNode, valueNode, fmt.Errorf("map build failed: reference cannot be found: %s",
							err.Error())
					}
				}
			}
		}
	}
	if valueNode != nil {
		var currentLabelNode *yaml.Node
		valueMap := make(map[KeyReference[string]]ValueReference[PT])

		bChan := make(chan mappingResult[PT])
		eChan := make(chan error)

		buildMap := func(label *yaml.Node, value *yaml.Node, c chan mappingResult[PT], ec chan<- error, ref string) {
			var n PT = new(N)
			_ = BuildModel(value, n)
			err := n.Build(value, idx)
			if err != nil {
				ec <- err
				return
			}

			//isRef := false
			if ref != "" {
				//isRef = true
				SetReference(n, ref)
			}

			c <- mappingResult[PT]{
				k: KeyReference[string]{
					KeyNode: label,
					Value:   label.Value,
				},
				v: ValueReference[PT]{
					Value:     n,
					ValueNode: value,
					//IsReference: isRef,
					Reference: ref,
				},
			}
		}

		totalKeys := 0
		for i, en := range valueNode.Content {
			referenceValue = ""
			if i%2 == 0 {
				currentLabelNode = en
				continue
			}
			// check our valueNode isn't a reference still.
			if h, _, refVal := utils.IsNodeRefValue(en); h {
				ref, err := LocateRefNode(en, idx)
				if ref != nil {
					en = ref
					referenceValue = refVal
					if err != nil {
						circError = err
					}
				} else {
					if err != nil {
						return nil, labelNode, valueNode, fmt.Errorf("flat map build failed: reference cannot be found: %s",
							err.Error())
					}
				}
			}

			if !extensions {
				if strings.HasPrefix(currentLabelNode.Value, "x-") {
					continue // yo, don't pay any attention to extensions, not here anyway.
				}
			}
			totalKeys++
			go buildMap(currentLabelNode, en, bChan, eChan, referenceValue)
		}

		completedKeys := 0
		for completedKeys < totalKeys {
			select {
			case err := <-eChan:
				return valueMap, labelNode, valueNode, err
			case res := <-bChan:
				completedKeys++
				valueMap[res.k] = res.v
			}
		}
		if circError != nil && !idx.AllowCircularReferenceResolving() {
			return valueMap, labelNode, valueNode, circError
		}
		return valueMap, labelNode, valueNode, nil
	}
	return nil, labelNode, valueNode, nil
}

// ExtractMap will extract a map of KeyReference and ValueReference from a root yaml.Node. The 'label' is
// used to locate the node to be extracted from the root node supplied.
//
// The second return value is the yaml.Node found for the 'label' and the third return value is the yaml.Node
// found for the value extracted from the label node.
func ExtractMap[PT Buildable[N], N any](
	label string,
	root *yaml.Node,
	idx *index.SpecIndex,
) (map[KeyReference[string]]ValueReference[PT], *yaml.Node, *yaml.Node, error) {
	return ExtractMapExtensions[PT, N](label, root, idx, false)
}

// ExtractExtensions will extract any 'x-' prefixed key nodes from a root node into a map. Requirements have been pre-cast:
//
// Maps
//
//	map[string]interface{} for maps
//
// Slices
//
//	[]interface{}
//
// int, float, bool, string
//
//	int64, float64, bool, string
func ExtractExtensions(root *yaml.Node) map[KeyReference[string]]ValueReference[any] {
	extensions := utils.FindExtensionNodes(root.Content)
	extensionMap := make(map[KeyReference[string]]ValueReference[any])
	for _, ext := range extensions {
		if utils.IsNodeMap(ext.Value) {
			var v interface{}
			_ = ext.Value.Decode(&v)
			extensionMap[KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = ValueReference[any]{Value: v, ValueNode: ext.Value}
		}
		if utils.IsNodeStringValue(ext.Value) {
			extensionMap[KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = ValueReference[any]{Value: ext.Value.Value, ValueNode: ext.Value}
		}
		if utils.IsNodeFloatValue(ext.Value) {
			fv, _ := strconv.ParseFloat(ext.Value.Value, 64)
			extensionMap[KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = ValueReference[any]{Value: fv, ValueNode: ext.Value}
		}
		if utils.IsNodeIntValue(ext.Value) {
			iv, _ := strconv.ParseInt(ext.Value.Value, 10, 64)
			extensionMap[KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = ValueReference[any]{Value: iv, ValueNode: ext.Value}
		}
		if utils.IsNodeBoolValue(ext.Value) {
			bv, _ := strconv.ParseBool(ext.Value.Value)
			extensionMap[KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = ValueReference[any]{Value: bv, ValueNode: ext.Value}
		}
		if utils.IsNodeArray(ext.Value) {
			var v []interface{}
			_ = ext.Value.Decode(&v)
			extensionMap[KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = ValueReference[any]{Value: v, ValueNode: ext.Value}
		}
	}
	return extensionMap
}

// AreEqual returns true if two Hashable objects are equal or not.
func AreEqual(l, r Hashable) bool {
	if l == nil || r == nil {
		return false
	}
	return l.Hash() == r.Hash()
}

// GenerateHashString will generate a SHA36 hash of any object passed in. If the object is Hashable
// then the underlying Hash() method will be called.
func GenerateHashString(v any) string {
	if h, ok := v.(Hashable); ok {
		if h != nil {
			return fmt.Sprintf(HASH, h.Hash())
		}
	}
	return fmt.Sprintf(HASH, sha256.Sum256([]byte(fmt.Sprint(v))))
}

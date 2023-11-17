// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// FindItemInMap accepts a string key and a collection of KeyReference[string] and ValueReference[T]. Every
// KeyReference will have its value checked against the string key and if there is a match, it will be returned.
func FindItemInMap[T any](item string, collection map[KeyReference[string]]ValueReference[T]) *ValueReference[T] {
	for n, o := range collection {
		if n.Value == item {
			return &o
		}
		if strings.EqualFold(item, n.Value) {
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

func LocateRefNodeWithContext(ctx context.Context, root *yaml.Node, idx *index.SpecIndex) (*yaml.Node, *index.SpecIndex, error, context.Context) {

	if rf, _, rv := utils.IsNodeRefValue(root); rf {

		if rv == "" {
			return nil, nil, fmt.Errorf("reference at line %d, column %d is empty, it cannot be resolved",
				root.Line, root.Column), ctx
		}

		// run through everything and return as soon as we find a match.
		// this operates as fast as possible as ever
		collections := generateIndexCollection(idx)
		var found map[string]*index.Reference
		for _, collection := range collections {
			found = collection()
			if found != nil && found[rv] != nil {

				// if this is a ref node, we need to keep diving
				// until we hit something that isn't a ref.
				if jh, _, _ := utils.IsNodeRefValue(found[rv].Node); jh {
					// if this node is circular, stop drop and roll.
					if !IsCircular(found[rv].Node, idx) {
						return LocateRefNodeWithContext(ctx, found[rv].Node, idx)
					} else {
						return found[rv].Node, idx, fmt.Errorf("circular reference '%s' found during lookup at line "+
							"%d, column %d, It cannot be resolved",
							GetCircularReferenceResult(found[rv].Node, idx).GenerateJourneyPath(),
							found[rv].Node.Line,
							found[rv].Node.Column), ctx
					}
				}
				return utils.NodeAlias(found[rv].Node), idx, nil, ctx
			}
		}

		// perform a search for the reference in the index
		// extract the correct root
		specPath := idx.GetSpecAbsolutePath()
		if ctx.Value(index.CurrentPathKey) != nil {
			specPath = ctx.Value(index.CurrentPathKey).(string)
		}

		explodedRefValue := strings.Split(rv, "#")
		if len(explodedRefValue) == 2 {
			if !strings.HasPrefix(explodedRefValue[0], "http") {

				if !filepath.IsAbs(explodedRefValue[0]) {

					if strings.HasPrefix(specPath, "http") {
						u, _ := url.Parse(specPath)
						p := ""
						if u.Path != "" {
							p = filepath.Dir(u.Path)
						}
						u.Path = filepath.Join(p, explodedRefValue[0])
						u.Fragment = ""
						rv = fmt.Sprintf("%s#%s", u.String(), explodedRefValue[1])

					} else {
						if specPath != "" {
							var abs string
							if explodedRefValue[0] == "" {
								abs = specPath
							} else {
								abs, _ = filepath.Abs(filepath.Join(filepath.Dir(specPath), explodedRefValue[0]))
							}
							rv = fmt.Sprintf("%s#%s", abs, explodedRefValue[1])
						} else {

							// check for a config baseURL and use that if it exists.
							if idx.GetConfig().BaseURL != nil {

								u := *idx.GetConfig().BaseURL
								p := ""
								if u.Path != "" {
									p = filepath.Dir(u.Path)
								}
								u.Path = filepath.Join(p, explodedRefValue[0])
								rv = fmt.Sprintf("%s#%s", u.String(), explodedRefValue[1])
							}
						}
					}
				}
			}
		} else {

			if !strings.HasPrefix(explodedRefValue[0], "http") {

				if !filepath.IsAbs(explodedRefValue[0]) {

					if strings.HasPrefix(specPath, "http") {
						u, _ := url.Parse(specPath)
						p := filepath.Dir(u.Path)
						abs, _ := filepath.Abs(filepath.Join(p, rv))
						u.Path = abs
						rv = u.String()

					} else {
						if specPath != "" {

							abs, _ := filepath.Abs(filepath.Join(filepath.Dir(specPath), rv))
							rv = abs

						} else {

							// check for a config baseURL and use that if it exists.
							if idx.GetConfig().BaseURL != nil {
								u := *idx.GetConfig().BaseURL
								abs, _ := filepath.Abs(filepath.Join(u.Path, rv))
								u.Path = abs
								rv = u.String()
							}
						}
					}
				}
			}
		}

		foundRef, fIdx, newCtx := idx.SearchIndexForReferenceWithContext(ctx, rv)
		if foundRef != nil {
			return utils.NodeAlias(foundRef.Node), fIdx, nil, newCtx
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
						return utils.NodeAlias(nodes[0]), idx, nil, ctx
					}
				}
			}
		}
		return nil, idx, fmt.Errorf("reference '%s' at line %d, column %d was not found",
			rv, root.Line, root.Column), ctx
	}
	return nil, idx, nil, ctx

}

// LocateRefNode will perform a complete lookup for a $ref node. This function searches the entire index for
// the reference being supplied. If there is a match found, the reference *yaml.Node is returned.
func LocateRefNode(root *yaml.Node, idx *index.SpecIndex) (*yaml.Node, *index.SpecIndex, error) {
	r, i, e, _ := LocateRefNodeWithContext(context.Background(), root, idx)
	return r, i, e
}

// ExtractObjectRaw will extract a typed Buildable[N] object from a root yaml.Node. The 'raw' aspect is
// that there is no NodeReference wrapper around the result returned, just the raw object.
func ExtractObjectRaw[T Buildable[N], N any](ctx context.Context, key, root *yaml.Node, idx *index.SpecIndex) (T, error, bool, string) {
	var circError error
	var isReference bool
	var referenceValue string
	root = utils.NodeAlias(root)
	if h, _, rv := utils.IsNodeRefValue(root); h {
		ref, fIdx, err, nCtx := LocateRefNodeWithContext(ctx, root, idx)
		if ref != nil {
			root = ref
			isReference = true
			referenceValue = rv
			idx = fIdx
			ctx = nCtx
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
	err = n.Build(ctx, key, root, idx)
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
func ExtractObject[T Buildable[N], N any](ctx context.Context, label string, root *yaml.Node, idx *index.SpecIndex) (NodeReference[T], error) {
	var ln, vn *yaml.Node
	var circError error
	var isReference bool
	var referenceValue string
	root = utils.NodeAlias(root)
	if rf, rl, refVal := utils.IsNodeRefValue(root); rf {
		ref, fIdx, err, nCtx := LocateRefNodeWithContext(ctx, root, idx)
		if ref != nil {
			vn = ref
			ln = rl
			isReference = true
			referenceValue = refVal
			idx = fIdx
			ctx = nCtx
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
				ref, fIdx, lerr, nCtx := LocateRefNodeWithContext(ctx, vn, idx)
				if ref != nil {
					vn = ref
					if fIdx != nil {
						idx = fIdx
					}
					ctx = nCtx
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
	err = n.Build(ctx, ln, vn, idx)
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
func ExtractArray[T Buildable[N], N any](ctx context.Context, label string, root *yaml.Node, idx *index.SpecIndex) ([]ValueReference[T],
	*yaml.Node, *yaml.Node, error,
) {
	var ln, vn *yaml.Node
	var circError error
	root = utils.NodeAlias(root)
	isRef := false
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		ref, fIdx, err, nCtx := LocateRefEnd(ctx, root, idx, 0)
		if ref != nil {
			isRef = true
			vn = ref
			ln = rl
			idx = fIdx
			ctx = nCtx
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
				ref, fIdx, err, nCtx := LocateRefEnd(ctx, vn, idx, 0)
				if ref != nil {
					isRef = true
					vn = ref
					idx = fIdx
					ctx = nCtx
					if err != nil {
						circError = err
					}
				} else {
					if err != nil {
						return []ValueReference[T]{}, nil, nil,
							fmt.Errorf("array build failed: reference cannot be found: %s",
								err.Error())
					}
				}
			}
		}
	}

	var items []ValueReference[T]
	if vn != nil && ln != nil {
		if !utils.IsNodeArray(vn) {

			if !isRef {
				return []ValueReference[T]{}, nil, nil,
					fmt.Errorf("array build failed, input is not an array, line %d, column %d", vn.Line, vn.Column)
			}
			// if this was pulled from a ref, but it's not a sequence, check the label and see if anything comes out,
			// and then check that is a sequence, if not, fail it.
			_, _, fvn := utils.FindKeyNodeFullTop(label, vn.Content)
			if fvn != nil {
				if !utils.IsNodeArray(vn) {
					return []ValueReference[T]{}, nil, nil,
						fmt.Errorf("array build failed, input is not an array, line %d, column %d", vn.Line, vn.Column)
				}
			}
		}
		for _, node := range vn.Content {
			localReferenceValue := ""
			foundCtx := ctx
			foundIndex := idx

			if rf, _, rv := utils.IsNodeRefValue(node); rf {
				refg, fIdx, err, nCtx := LocateRefEnd(ctx, node, idx, 0)
				if refg != nil {
					node = refg
					localReferenceValue = rv
					foundIndex = fIdx
					foundCtx = nCtx
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
			berr := n.Build(foundCtx, ln, node, foundIndex)
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
	ctx context.Context,
	root *yaml.Node,
	idx *index.SpecIndex,
	includeExtensions bool,
) (map[KeyReference[string]]ValueReference[PT], error) {
	valueMap := make(map[KeyReference[string]]ValueReference[PT])
	var circError error
	if utils.IsNodeMap(root) {
		var currentKey *yaml.Node
		skip := false
		rlen := len(root.Content)

		for i := 0; i < rlen; i++ {
			node := root.Content[i]
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

			if currentKey.Tag == "!!merge" && currentKey.Value == "<<" {
				root.Content = append(root.Content, utils.NodeAlias(node).Content...)
				rlen = len(root.Content)
				currentKey = nil
				continue
			}
			node = utils.NodeAlias(node)

			foundIndex := idx
			foundContext := ctx

			var isReference bool
			var referenceValue string
			// if value is a reference, we have to look it up in the index!
			if h, _, rv := utils.IsNodeRefValue(node); h {
				ref, fIdx, err, nCtx := LocateRefNodeWithContext(ctx, node, idx)
				if ref != nil {
					node = ref
					isReference = true
					referenceValue = rv
					if fIdx != nil {
						foundIndex = fIdx
					}
					foundContext = nCtx
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
			berr := n.Build(foundContext, currentKey, node, foundIndex)
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
	ctx context.Context,
	root *yaml.Node,
	idx *index.SpecIndex,
) (map[KeyReference[string]]ValueReference[PT], error) {
	return ExtractMapNoLookupExtensions[PT, N](ctx, root, idx, false)
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
	ctx context.Context,
	label string,
	root *yaml.Node,
	idx *index.SpecIndex,
	extensions bool,
) (map[KeyReference[string]]ValueReference[PT], *yaml.Node, *yaml.Node, error) {
	var referenceValue string
	var labelNode, valueNode *yaml.Node
	var circError error
	root = utils.NodeAlias(root)
	if rf, rl, rv := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref, fIdx, err, fCtx := LocateRefNodeWithContext(ctx, root, idx)
		if ref != nil {
			valueNode = ref
			labelNode = rl
			referenceValue = rv
			ctx = fCtx
			idx = fIdx
			if err != nil {
				circError = err
			}
		} else {
			return nil, labelNode, valueNode, fmt.Errorf("map build failed: reference cannot be found: %s",
				root.Content[1].Value)
		}
	} else {
		_, labelNode, valueNode = utils.FindKeyNodeFull(label, root.Content)
		valueNode = utils.NodeAlias(valueNode)
		if valueNode != nil {
			if h, _, rvt := utils.IsNodeRefValue(valueNode); h {
				ref, fIdx, err, nCtx := LocateRefNodeWithContext(ctx, valueNode, idx)
				if ref != nil {
					valueNode = ref
					referenceValue = rvt
					idx = fIdx
					ctx = nCtx
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

		buildMap := func(nctx context.Context, label *yaml.Node, value *yaml.Node, c chan mappingResult[PT], ec chan<- error, ref string, fIdx *index.SpecIndex) {
			var n PT = new(N)
			value = utils.NodeAlias(value)
			_ = BuildModel(value, n)
			err := n.Build(nctx, label, value, fIdx)
			if err != nil {
				ec <- err
				return
			}

			if ref != "" {
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
					Reference: ref,
				},
			}
		}

		totalKeys := 0
		for i, en := range valueNode.Content {
			en = utils.NodeAlias(en)
			referenceValue = ""
			if i%2 == 0 {
				currentLabelNode = en
				continue
			}

			foundIndex := idx
			foundContext := ctx

			// check our valueNode isn't a reference still.
			if h, _, refVal := utils.IsNodeRefValue(en); h {
				ref, fIdx, err, nCtx := LocateRefNodeWithContext(ctx, en, idx)
				if ref != nil {
					en = ref
					referenceValue = refVal
					if fIdx != nil {
						foundIndex = fIdx
					}
					foundContext = nCtx
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
			go buildMap(foundContext, currentLabelNode, en, bChan, eChan, referenceValue, foundIndex)
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
	ctx context.Context,
	label string,
	root *yaml.Node,
	idx *index.SpecIndex,
) (map[KeyReference[string]]ValueReference[PT], *yaml.Node, *yaml.Node, error) {
	return ExtractMapExtensions[PT, N](ctx, label, root, idx, false)
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
	root = utils.NodeAlias(root)
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
	if reflect.ValueOf(l).IsNil() || reflect.ValueOf(r).IsNil() {
		return false
	}
	return l.Hash() == r.Hash()
}

// GenerateHashString will generate a SHA36 hash of any object passed in. If the object is Hashable
// then the underlying Hash() method will be called.
func GenerateHashString(v any) string {
	if v == nil {
		return ""
	}
	if h, ok := v.(Hashable); ok {
		if h != nil {
			return fmt.Sprintf(HASH, h.Hash())
		}
	}
	// if we get here, we're a primitive, check if we're a pointer and de-point
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		v = reflect.ValueOf(v).Elem().Interface()
	}
	return fmt.Sprintf(HASH, sha256.Sum256([]byte(fmt.Sprint(v))))
}

// LocateRefEnd will perform a complete lookup for a $ref node. This function searches the entire index for
// the reference being supplied. If there is a match found, the reference *yaml.Node is returned.
// the function operates recursively and will keep iterating through references until it finds a non-reference
// node.
func LocateRefEnd(ctx context.Context, root *yaml.Node, idx *index.SpecIndex, depth int) (*yaml.Node, *index.SpecIndex, error, context.Context) {
	depth++
	if depth > 100 {
		return nil, nil, fmt.Errorf("reference resolution depth exceeded, possible circular reference"), ctx
	}
	ref, fIdx, err, nCtx := LocateRefNodeWithContext(ctx, root, idx)
	if err != nil {
		return ref, fIdx, err, nCtx
	}
	if rf, _, _ := utils.IsNodeRefValue(ref); rf {
		return LocateRefEnd(nCtx, ref, fIdx, depth)
	} else {
		return ref, fIdx, err, nCtx
	}
}

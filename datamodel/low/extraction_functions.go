// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"fmt"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

func FindItemInMap[T any](item string, collection map[KeyReference[string]]ValueReference[T]) *ValueReference[T] {
	for n, o := range collection {
		if n.Value == item {
			return &o
		}
	}
	return nil
}

func LocateRefNode(root *yaml.Node, idx *index.SpecIndex) (*yaml.Node, error) {
	if rf, _, rv := utils.IsNodeRefValue(root); rf {
		// run through everything and return as soon as we find a match.
		// this operates as fast as possible as ever
		collections := []func() map[string]*index.Reference{
			idx.GetAllSchemas,
			idx.GetMappedReferences,
			idx.GetAllExternalDocuments,
			idx.GetAllParameters,
			idx.GetAllHeaders,
			idx.GetAllCallbacks,
			idx.GetAllLinks,
			idx.GetAllExternalDocuments,
			idx.GetAllExamples,
			idx.GetAllRequestBodies,
			idx.GetAllResponses,
			idx.GetAllSecuritySchemes,
		}

		// if there are any external indexes being used by remote
		// documents, then we need to search through them also.
		externalIndexes := idx.GetAllExternalIndexes()
		if len(externalIndexes) > 0 {
			var extCollection []func() map[string]*index.Reference
			for _, extIndex := range externalIndexes {
				extCollection = []func() map[string]*index.Reference{
					extIndex.GetAllSchemas,
					extIndex.GetMappedReferences,
					extIndex.GetAllExternalDocuments,
					extIndex.GetAllParameters,
					extIndex.GetAllHeaders,
					extIndex.GetAllCallbacks,
					extIndex.GetAllLinks,
					extIndex.GetAllExternalDocuments,
					extIndex.GetAllExamples,
					extIndex.GetAllRequestBodies,
					extIndex.GetAllResponses,
					extIndex.GetAllSecuritySchemes,
				}
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
						Log.Error("circular reference found during lookup, and will remain un-resolved.",
							zap.Int("line", found[rv].Node.Line),
							zap.Int("column", found[rv].Node.Column),
							zap.String("reference", found[rv].Definition),
							zap.String("journey",
								GetCircularReferenceResult(found[rv].Node, idx).GenerateJourneyPath()))

						return found[rv].Node, fmt.Errorf("circular reference '%s' found during lookup at line %d, column %d, "+
							"It cannot be resolved",
							GetCircularReferenceResult(found[rv].Node, idx).GenerateJourneyPath(),
							found[rv].Node.Line,
							found[rv].Node.Column)
					}
				}
				return found[rv].Node, nil
			}
		}

		// cant be found? last resort is to try a path lookup
		cleaned := strings.ReplaceAll(rv, "#/paths/", "")
		cleaned = strings.ReplaceAll(cleaned, "/", ".")
		cleaned = strings.ReplaceAll(cleaned, "~1", "/")
		yamlPath := fmt.Sprintf("$.paths.%s", cleaned)
		path, err := yamlpath.NewPath(yamlPath)
		if err == nil {
			nodes, fErr := path.Find(idx.GetRootNode())
			if fErr == nil {
				if len(nodes) > 0 {

					if jh, _, _ := utils.IsNodeRefValue(nodes[0]); jh {
						if !IsCircular(nodes[0], idx) {
							return LocateRefNode(nodes[0], idx)
						} else {
							Log.Error("circular reference found during lookup, and will remain un-resolved.",
								zap.Int("column", nodes[0].Column),
								zap.String("reference", yamlPath),
								zap.String("journey",
									GetCircularReferenceResult(nodes[0], idx).GenerateJourneyPath()))
							if !idx.AllowCircularReferenceResolving() {
								return found[rv].Node, fmt.Errorf(
									"circular reference '%s' found during lookup at line %d, column %d, "+
										"It cannot be resolved",
									GetCircularReferenceResult(nodes[0], idx).GenerateJourneyPath(),
									nodes[0].Line,
									nodes[0].Column)
							}
						}
					}
					return nodes[0], nil
				}
			}
		}
		return nil, fmt.Errorf("reference '%s' at line %d, column %d was not found", root.Value, root.Line, root.Column)
	}
	return nil, nil
}

func ExtractObjectRaw[T Buildable[N], N any](root *yaml.Node, idx *index.SpecIndex) (T, error) {
	var circError error
	if h, _, _ := utils.IsNodeRefValue(root); h {
		ref, err := LocateRefNode(root, idx)
		if ref != nil {
			root = ref
			if err != nil {
				circError = err
			}
		} else {
			if err != nil {
				return nil, fmt.Errorf("object extraciton failed: %s", err.Error())
			}
		}
	}
	var n T = new(N)
	err := BuildModel(root, n)
	if err != nil {
		return n, err
	}
	err = n.Build(root, idx)
	if err != nil {
		return n, err
	}
	if circError != nil && !idx.AllowCircularReferenceResolving() {
		return n, circError
	}
	return n, nil
}

func ExtractObject[T Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) (NodeReference[T], error) {
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
			if err != nil {
				return NodeReference[T]{}, fmt.Errorf("object extraciton failed: %s", err.Error())
			}
		}
	} else {
		_, ln, vn = utils.FindKeyNodeFull(label, root.Content)
		if vn != nil {
			if h, _, _ := utils.IsNodeRefValue(vn); h {
				ref, lerr := LocateRefNode(vn, idx)
				if ref != nil {
					vn = ref
					if lerr != nil {
						circError = lerr
					}
				} else {
					if lerr != nil {
						return NodeReference[T]{}, fmt.Errorf("object extraciton failed: %s", lerr.Error())
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
	res := NodeReference[T]{
		Value:     n,
		KeyNode:   ln,
		ValueNode: vn,
	}
	if circError != nil && !idx.AllowCircularReferenceResolving() {
		return res, err
	}
	return res, nil
}

func ExtractArray[T Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) ([]ValueReference[T],
	*yaml.Node, *yaml.Node, error) {
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
		_, ln, vn = utils.FindKeyNodeFull(label, root.Content)
		if vn != nil {
			if h, _, _ := utils.IsNodeRefValue(vn); h {
				ref, err := LocateRefNode(vn, idx)
				if ref != nil {
					vn = ref
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
			if rf, _, _ := utils.IsNodeRefValue(node); rf {
				ref, err := LocateRefNode(node, idx)
				if ref != nil {
					node = ref
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
			items = append(items, ValueReference[T]{
				Value:     n,
				ValueNode: node,
			})
		}
	}
	if circError != nil && !idx.AllowCircularReferenceResolving() {
		return items, ln, vn, circError
	}
	return items, ln, vn, nil
}

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

func ExtractMapFlatNoLookup[PT Buildable[N], N any](root *yaml.Node, idx *index.SpecIndex) (map[KeyReference[string]]ValueReference[PT], error) {
	valueMap := make(map[KeyReference[string]]ValueReference[PT])
	var circError error
	if utils.IsNodeMap(root) {
		var currentKey *yaml.Node
		skip := false
		for i, node := range root.Content {
			if strings.HasPrefix(strings.ToLower(node.Value), "x-") {
				skip = true
				continue
			}
			if skip {
				skip = false
				continue
			}
			if i%2 == 0 {
				currentKey = node
				continue
			}
			// if value is a reference, we have to look it up in the index!
			if h, _, _ := utils.IsNodeRefValue(node); h {
				ref, err := LocateRefNode(node, idx)
				if ref != nil {
					node = ref
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
			valueMap[KeyReference[string]{
				Value:   currentKey.Value,
				KeyNode: currentKey,
			}] = ValueReference[PT]{
				Value:     n,
				ValueNode: node,
			}
		}
	}
	if circError != nil && !idx.AllowCircularReferenceResolving() {
		return valueMap, circError
	}
	return valueMap, nil
}

type mappingResult[T any] struct {
	k KeyReference[string]
	v ValueReference[T]
}

func ExtractMapFlat[PT Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) (map[KeyReference[string]]ValueReference[PT], *yaml.Node, *yaml.Node, error) {
	var labelNode, valueNode *yaml.Node
	var circError error
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref, err := LocateRefNode(root, idx)
		if ref != nil {
			valueNode = ref
			labelNode = rl
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
			if h, _, _ := utils.IsNodeRefValue(valueNode); h {
				ref, err := LocateRefNode(valueNode, idx)
				if ref != nil {
					valueNode = ref
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

		var buildMap = func(label *yaml.Node, value *yaml.Node, c chan mappingResult[PT], ec chan<- error) {
			var n PT = new(N)
			_ = BuildModel(value, n)
			err := n.Build(value, idx)
			if err != nil {
				ec <- err
				return
			}
			c <- mappingResult[PT]{
				k: KeyReference[string]{
					KeyNode: label,
					Value:   label.Value,
				},
				v: ValueReference[PT]{
					Value:     n,
					ValueNode: value,
				},
			}
		}

		totalKeys := 0
		for i, en := range valueNode.Content {
			if i%2 == 0 {
				currentLabelNode = en
				continue
			}
			// check our valueNode isn't a reference still.
			if h, _, _ := utils.IsNodeRefValue(en); h {
				ref, err := LocateRefNode(en, idx)
				if ref != nil {
					en = ref
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

			if strings.HasPrefix(strings.ToLower(currentLabelNode.Value), "x-") {
				continue // yo, don't pay any attention to extensions, not here anyway.
			}
			totalKeys++
			go buildMap(currentLabelNode, en, bChan, eChan)
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
		return valueMap, labelNode, valueNode, nil
	}
	if circError != nil && !idx.AllowCircularReferenceResolving() {
		return nil, labelNode, valueNode, circError
	}
	return nil, labelNode, valueNode, nil
}

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

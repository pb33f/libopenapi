// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"fmt"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
	"sync"
)

func FindItemInMap[T any](item string, collection map[KeyReference[string]]ValueReference[T]) *ValueReference[T] {
	for n, o := range collection {
		if n.Value == item {
			return &o
		}
	}
	return nil
}

var mapLock sync.Mutex

func LocateRefNode(root *yaml.Node, idx *index.SpecIndex) *yaml.Node {
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
				return found[rv].Node
			}
		}

		// cant be found? last resort is to try a path lookup
		cleaned := strings.ReplaceAll(rv, "#/paths/", "")
		cleaned = strings.ReplaceAll(cleaned, "/", ".")
		cleaned = strings.ReplaceAll(cleaned, "~1", "/")
		path, err := yamlpath.NewPath(fmt.Sprintf("$.paths.%s", cleaned))
		if err == nil {
			nodes, fErr := path.Find(idx.GetRootNode())
			if fErr == nil {
				if len(nodes) > 0 {
					return nodes[0]
				}
			}
		}
	}
	return nil
}

func ExtractObjectRaw[T Buildable[N], N any](root *yaml.Node, idx *index.SpecIndex) (T, error) {

	if h, _, _ := utils.IsNodeRefValue(root); h {
		ref := LocateRefNode(root, idx)
		if ref != nil {
			root = ref
		} else {
			return nil, fmt.Errorf("object extraction failed: reference cannot be found: %s, line %d, col %d",
				root.Content[1].Value, root.Content[1].Line, root.Content[1].Column)
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
	return n, nil
}

func ExtractObject[T Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) (NodeReference[T], error) {
	var ln, vn *yaml.Node
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref := LocateRefNode(root, idx)
		if ref != nil {
			vn = ref
			ln = rl
		} else {
			return NodeReference[T]{}, fmt.Errorf("object build failed: reference cannot be found: %s",
				root.Content[1].Value)
		}
	} else {
		_, ln, vn = utils.FindKeyNodeFull(label, root.Content)
		if vn != nil {
			if h, _, _ := utils.IsNodeRefValue(vn); h {
				ref := LocateRefNode(vn, idx)
				if ref != nil {
					vn = ref
				} else {
					return NodeReference[T]{}, fmt.Errorf("object build failed: reference cannot be found: %s",
						root.Content[1].Value)
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
	return NodeReference[T]{
		Value:     n,
		KeyNode:   ln,
		ValueNode: vn,
	}, nil
}

func ExtractArray[T Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) ([]ValueReference[T],
	*yaml.Node, *yaml.Node, error) {
	var ln, vn *yaml.Node
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref := LocateRefNode(root, idx)
		if ref != nil {
			vn = ref
			ln = rl
		} else {
			return []ValueReference[T]{}, nil, nil, fmt.Errorf("array build failed: reference cannot be found: %s",
				root.Content[1].Value)
		}
	} else {
		_, ln, vn = utils.FindKeyNodeFull(label, root.Content)
		if vn != nil {
			if h, _, _ := utils.IsNodeRefValue(vn); h {
				ref := LocateRefNode(vn, idx)
				if ref != nil {
					vn = ref
				} else {
					return []ValueReference[T]{}, nil, nil, fmt.Errorf("array build failed: reference cannot be found: %s",
						root.Content[1].Value)
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
				ref := LocateRefNode(node, idx)
				if ref != nil {
					node = ref
				} else {
					return []ValueReference[T]{}, nil, nil, fmt.Errorf("array build failed: reference cannot be found: %s",
						root.Content[1].Value)
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
				ref := LocateRefNode(node, idx)
				if ref != nil {
					node = ref
				} else {
					return nil, fmt.Errorf("map build failed: reference cannot be found: %s", root.Content[1].Value)
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
	return valueMap, nil
}

func ExtractMapFlat[PT Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) (map[KeyReference[string]]ValueReference[PT], *yaml.Node, *yaml.Node, error) {
	var labelNode, valueNode *yaml.Node
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref := LocateRefNode(root, idx)
		if ref != nil {
			valueNode = ref
			labelNode = rl
		} else {
			return nil, labelNode, valueNode, fmt.Errorf("map build failed: reference cannot be found: %s",
				root.Content[1].Value)
		}
	} else {
		_, labelNode, valueNode = utils.FindKeyNodeFull(label, root.Content)
		if valueNode != nil {
			if h, _, _ := utils.IsNodeRefValue(valueNode); h {
				ref := LocateRefNode(valueNode, idx)
				if ref != nil {
					valueNode = ref
				} else {
					return nil, labelNode, valueNode, fmt.Errorf("map build failed: reference cannot be found: %s",
						root.Content[1].Value)
				}
			}
		}
	}
	if valueNode != nil {
		var currentLabelNode *yaml.Node
		valueMap := make(map[KeyReference[string]]ValueReference[PT])
		for i, en := range valueNode.Content {
			if i%2 == 0 {
				currentLabelNode = en
				continue
			}

			// check our valueNode isn't a reference still.
			if h, _, _ := utils.IsNodeRefValue(en); h {
				ref := LocateRefNode(en, idx)
				if ref != nil {
					en = ref
				} else {
					return nil, labelNode, valueNode, fmt.Errorf("flat map build failed: reference cannot be found: %s",
						root.Content[1].Value)
				}
			}

			if strings.HasPrefix(strings.ToLower(currentLabelNode.Value), "x-") {
				continue // yo, don't pay any attention to extensions, not here anyway.
			}
			var n PT = new(N)
			err := BuildModel(en, n)
			if err != nil {
				return nil, labelNode, valueNode, err
			}
			berr := n.Build(en, idx)
			if berr != nil {
				return nil, labelNode, valueNode, berr
			}
			valueMap[KeyReference[string]{
				Value:   currentLabelNode.Value,
				KeyNode: currentLabelNode,
			}] = ValueReference[PT]{
				Value:     n,
				ValueNode: en,
			}
		}
		return valueMap, labelNode, valueNode, nil
	}
	return nil, labelNode, valueNode, nil
}

func ExtractMap[PT Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) (map[KeyReference[string]]map[KeyReference[string]]ValueReference[PT], error) {
	var labelNode, valueNode *yaml.Node
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref := LocateRefNode(root, idx)
		if ref != nil {
			valueNode = ref
			labelNode = rl
		} else {
			return nil, fmt.Errorf("map build failed: reference cannot be found: %s", root.Content[1].Value)
		}
	} else {
		_, labelNode, valueNode = utils.FindKeyNodeFull(label, root.Content)
		if valueNode != nil {
			if h, _, _ := utils.IsNodeRefValue(valueNode); h {
				ref := LocateRefNode(valueNode, idx)
				if ref != nil {
					valueNode = ref
				} else {
					return nil, fmt.Errorf("map build failed: reference cannot be found: %s",
						root.Content[1].Value)
				}
			}
		}
	}

	if valueNode != nil {
		var currentLabelNode *yaml.Node
		valueMap := make(map[KeyReference[string]]ValueReference[PT])
		for i, en := range valueNode.Content {
			if i%2 == 0 {
				currentLabelNode = en
				continue
			}
			if strings.HasPrefix(strings.ToLower(currentLabelNode.Value), "x-") {
				continue // yo, don't pay any attention to extensions, not here anyway.
			}

			// check our valueNode isn't a reference still.
			if h, _, _ := utils.IsNodeRefValue(en); h {
				ref := LocateRefNode(en, idx)
				if ref != nil {
					en = ref
				} else {
					return nil, fmt.Errorf("map build failed: reference cannot be found: %s",
						root.Content[1].Value)
				}
			}

			var n PT = new(N)
			err := BuildModel(en, n)
			if err != nil {
				return nil, err
			}
			berr := n.Build(en, idx)
			if berr != nil {
				return nil, berr
			}
			valueMap[KeyReference[string]{
				Value:   currentLabelNode.Value,
				KeyNode: currentLabelNode,
			}] = ValueReference[PT]{
				Value:     n,
				ValueNode: en,
			}
		}
		resMap := make(map[KeyReference[string]]map[KeyReference[string]]ValueReference[PT])
		resMap[KeyReference[string]{
			Value:   labelNode.Value,
			KeyNode: labelNode,
		}] = valueMap
		return resMap, nil
	}
	return nil, nil
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

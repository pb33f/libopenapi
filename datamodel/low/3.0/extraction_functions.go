package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
	"sync"
)

var KnownSchemas map[string]low.NodeReference[*Schema]

func init() {
	KnownSchemas = make(map[string]low.NodeReference[*Schema])

}

func FindItemInMap[T any](item string, collection map[low.KeyReference[string]]low.ValueReference[T]) *low.ValueReference[T] {
	for n, o := range collection {
		if n.Value == item {
			return &o
		}
	}
	return nil
}

func ExtractSchema(root *yaml.Node, idx *index.SpecIndex) (*low.NodeReference[*Schema], error) {
	var schLabel, schNode *yaml.Node
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref := LocateRefNode(root, idx)
		if ref != nil {
			schNode = ref
			schLabel = rl
		}
	} else {
		_, schLabel, schNode = utils.FindKeyNodeFull(SchemaLabel, root.Content)
		if schNode != nil {
			if h, _, _ := utils.IsNodeRefValue(schNode); h {
				ref := LocateRefNode(schNode, idx)
				if ref != nil {
					schNode = ref
				}
			}
		}
	}

	if schNode != nil {
		var schema Schema
		err := BuildModel(schNode, &schema)
		if err != nil {
			return nil, err
		}
		err = schema.Build(schNode, idx)
		if err != nil {
			return nil, err
		}
		return &low.NodeReference[*Schema]{Value: &schema, KeyNode: schLabel, ValueNode: schNode}, nil
	}
	return nil, nil
}

var mapLock sync.Mutex

func LocateRefNode(root *yaml.Node, idx *index.SpecIndex) *yaml.Node {
	if rf, _, rv := utils.IsNodeRefValue(root); rf {
		found := idx.GetMappedReferences()
		if found != nil && found[rv] != nil {
			return found[rv].Node
		}
	}
	return nil
}

func ExtractObjectRaw[T low.Buildable[N], N any](root *yaml.Node, idx *index.SpecIndex) (T, error) {
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

func ExtractObject[T low.Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) (low.NodeReference[T], error) {
	var ln, vn *yaml.Node
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref := LocateRefNode(root, idx)
		if ref != nil {
			vn = ref
			ln = rl
		}
	} else {
		_, ln, vn = utils.FindKeyNodeFull(label, root.Content)
		if vn != nil {
			if h, _, _ := utils.IsNodeRefValue(vn); h {
				ref := LocateRefNode(vn, idx)
				if ref != nil {
					vn = ref
				}
			}
		}
	}
	var n T = new(N)
	err := BuildModel(vn, n)
	if err != nil {
		return low.NodeReference[T]{}, err
	}
	if ln == nil {
		return low.NodeReference[T]{}, nil
	}
	err = n.Build(vn, idx)
	if err != nil {
		return low.NodeReference[T]{}, err
	}
	return low.NodeReference[T]{
		Value:     n,
		KeyNode:   ln,
		ValueNode: vn,
	}, nil
}

func ExtractArray[T low.Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) ([]low.ValueReference[T], *yaml.Node, *yaml.Node, error) {
	var ln, vn *yaml.Node
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref := LocateRefNode(root, idx)
		if ref != nil {
			vn = ref
			ln = rl
		}
	} else {
		_, ln, vn = utils.FindKeyNodeFull(label, root.Content)
		if vn != nil {
			if h, _, _ := utils.IsNodeRefValue(vn); h {
				ref := LocateRefNode(vn, idx)
				if ref != nil {
					vn = ref
				}
			}
		}
	}
	var items []low.ValueReference[T]
	if vn != nil && ln != nil {
		for _, node := range vn.Content {
			if rf, _, _ := utils.IsNodeRefValue(node); rf {
				ref := LocateRefNode(node, idx)
				if ref != nil {
					node = ref
				}
			}
			var n T = new(N)
			err := BuildModel(node, n)
			if err != nil {
				return []low.ValueReference[T]{}, ln, vn, err
			}
			berr := n.Build(node, idx)
			if berr != nil {
				return nil, ln, vn, berr
			}
			items = append(items, low.ValueReference[T]{
				Value:     n,
				ValueNode: node,
			})
		}
	}
	return items, ln, vn, nil
}

func ExtractMapFlatNoLookup[PT low.Buildable[N], N any](root *yaml.Node, idx *index.SpecIndex) (map[low.KeyReference[string]]low.ValueReference[PT], error) {
	valueMap := make(map[low.KeyReference[string]]low.ValueReference[PT])
	if utils.IsNodeMap(root) {
		var currentKey *yaml.Node
		for i, node := range root.Content {
			if i%2 == 0 {
				currentKey = node
				continue
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
			valueMap[low.KeyReference[string]{
				Value:   currentKey.Value,
				KeyNode: currentKey,
			}] = low.ValueReference[PT]{
				Value:     n,
				ValueNode: node,
			}
		}
	}
	return valueMap, nil
}

func ExtractMapFlat[PT low.Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) (map[low.KeyReference[string]]low.ValueReference[PT], *yaml.Node, *yaml.Node, error) {
	var labelNode, valueNode *yaml.Node
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref := LocateRefNode(root, idx)
		if ref != nil {
			valueNode = ref
			labelNode = rl
		}
	} else {
		_, labelNode, valueNode = utils.FindKeyNodeFull(label, root.Content)
		if valueNode != nil {
			if h, _, _ := utils.IsNodeRefValue(valueNode); h {
				ref := LocateRefNode(valueNode, idx)
				if ref != nil {
					valueNode = ref
				}
			}
		}
	}
	if valueNode != nil {
		var currentLabelNode *yaml.Node
		valueMap := make(map[low.KeyReference[string]]low.ValueReference[PT])
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
			valueMap[low.KeyReference[string]{
				Value:   currentLabelNode.Value,
				KeyNode: currentLabelNode,
			}] = low.ValueReference[PT]{
				Value:     n,
				ValueNode: en,
			}
		}
		return valueMap, labelNode, valueNode, nil
	}
	return nil, labelNode, valueNode, nil
}

func ExtractMap[PT low.Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) (map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[PT], error) {
	var labelNode, valueNode *yaml.Node
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref := LocateRefNode(root, idx)
		if ref != nil {
			valueNode = ref
			labelNode = rl
		}
	} else {
		_, labelNode, valueNode = utils.FindKeyNodeFull(label, root.Content)
	}

	if valueNode != nil {
		var currentLabelNode *yaml.Node
		valueMap := make(map[low.KeyReference[string]]low.ValueReference[PT])
		for i, en := range valueNode.Content {
			if i%2 == 0 {
				currentLabelNode = en
				continue
			}
			if strings.HasPrefix(strings.ToLower(currentLabelNode.Value), "x-") {
				continue // yo, don't pay any attention to extensions, not here anyway.
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
			valueMap[low.KeyReference[string]{
				Value:   currentLabelNode.Value,
				KeyNode: currentLabelNode,
			}] = low.ValueReference[PT]{
				Value:     n,
				ValueNode: en,
			}
		}
		resMap := make(map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[PT])
		resMap[low.KeyReference[string]{
			Value:   labelNode.Value,
			KeyNode: labelNode,
		}] = valueMap
		return resMap, nil
	}
	return nil, nil
}

func ExtractExtensions(root *yaml.Node) map[low.KeyReference[string]]low.ValueReference[any] {
	extensions := utils.FindExtensionNodes(root.Content)
	extensionMap := make(map[low.KeyReference[string]]low.ValueReference[any])
	for _, ext := range extensions {
		if utils.IsNodeMap(ext.Value) {
			var v interface{}
			_ = ext.Value.Decode(&v)
			extensionMap[low.KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.ValueReference[any]{Value: v, ValueNode: ext.Value}
		}
		if utils.IsNodeStringValue(ext.Value) {
			extensionMap[low.KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.ValueReference[any]{Value: ext.Value.Value, ValueNode: ext.Value}
		}
		if utils.IsNodeFloatValue(ext.Value) {
			fv, _ := strconv.ParseFloat(ext.Value.Value, 64)
			extensionMap[low.KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.ValueReference[any]{Value: fv, ValueNode: ext.Value}
		}
		if utils.IsNodeIntValue(ext.Value) {
			iv, _ := strconv.ParseInt(ext.Value.Value, 10, 64)
			extensionMap[low.KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.ValueReference[any]{Value: iv, ValueNode: ext.Value}
		}
		if utils.IsNodeBoolValue(ext.Value) {
			bv, _ := strconv.ParseBool(ext.Value.Value)
			extensionMap[low.KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.ValueReference[any]{Value: bv, ValueNode: ext.Value}
		}
		if utils.IsNodeArray(ext.Value) {
			var v []interface{}
			_ = ext.Value.Decode(&v)
			extensionMap[low.KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.ValueReference[any]{Value: v, ValueNode: ext.Value}
		}
	}
	return extensionMap
}

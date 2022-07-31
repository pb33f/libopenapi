package datamodel

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"reflect"
	"strconv"
	"sync"
)

func BuildModel(node *yaml.Node, model interface{}) error {

	if reflect.ValueOf(model).Type().Kind() != reflect.Pointer {
		return fmt.Errorf("cannot build model on non-pointer: %v", reflect.ValueOf(model).Type().Kind())
	}
	v := reflect.ValueOf(model).Elem()
	for i := 0; i < v.NumField(); i++ {

		fName := v.Type().Field(i).Name

		// we need to find a matching field in the YAML, the cases may be off, so take no chances.
		cases := []utils.Case{utils.PascalCase, utils.CamelCase, utils.ScreamingSnakeCase,
			utils.SnakeCase, utils.KebabCase, utils.RegularCase}

		var vn, kn *yaml.Node
		for _, tryCase := range cases {
			kn, vn = utils.FindKeyNode(utils.ConvertCase(fName, tryCase), node.Content)
			if vn != nil {
				break
			}
		}

		if vn == nil {
			// no point in going on.
			continue
		}

		field := v.FieldByName(fName)
		kind := field.Kind()
		switch kind {
		case reflect.Struct, reflect.Slice, reflect.Map, reflect.Pointer:
			err := SetField(field, vn, kn)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unable to parse unsupported type: %v", kind)
		}

	}

	return nil
}

func SetField(field reflect.Value, valueNode *yaml.Node, keyNode *yaml.Node) error {
	switch field.Type() {

	case reflect.TypeOf(map[string]low.ObjectReference{}):
		if valueNode != nil {
			if utils.IsNodeMap(valueNode) {
				if field.CanSet() {
					items := make(map[string]low.ObjectReference)
					var currentLabel string
					for i, sliceItem := range valueNode.Content {
						if i%2 == 0 {
							currentLabel = sliceItem.Value
							continue
						}
						var decoded map[string]interface{}
						err := sliceItem.Decode(&decoded)
						if err != nil {
							return err
						}
						items[currentLabel] = low.ObjectReference{
							Value:     decoded,
							ValueNode: sliceItem,
							KeyNode:   valueNode,
						}
					}
					field.Set(reflect.ValueOf(items))
				}
			}
		}
		break

	case reflect.TypeOf(map[string]low.NodeReference[string]{}):
		if valueNode != nil {
			if utils.IsNodeMap(valueNode) {
				if field.CanSet() {
					items := make(map[string]low.NodeReference[string])
					var currentLabel string
					for i, sliceItem := range valueNode.Content {
						if i%2 == 0 {
							currentLabel = sliceItem.Value
							continue
						}
						items[currentLabel] = low.NodeReference[string]{
							Value:     sliceItem.Value,
							ValueNode: sliceItem,
							KeyNode:   valueNode,
						}
					}
					field.Set(reflect.ValueOf(items))
				}
			}
		}
		break
	case reflect.TypeOf(low.ObjectReference{}):
		if valueNode != nil {
			var decoded map[string]interface{}
			err := valueNode.Decode(&decoded)
			if err != nil {
				return err
			}
			if utils.IsNodeMap(valueNode) {
				if field.CanSet() {
					or := low.ObjectReference{Value: decoded, ValueNode: valueNode}
					field.Set(reflect.ValueOf(or))
				}
			}
		}
		break
	case reflect.TypeOf([]low.ObjectReference{}):
		if valueNode != nil {
			if utils.IsNodeArray(valueNode) {
				if field.CanSet() {
					var items []low.ObjectReference
					for _, sliceItem := range valueNode.Content {
						var decoded map[string]interface{}
						err := sliceItem.Decode(&decoded)
						if err != nil {
							return err
						}
						items = append(items, low.ObjectReference{
							Value:     decoded,
							ValueNode: sliceItem,
							KeyNode:   valueNode,
						})
					}
					field.Set(reflect.ValueOf(items))
				}
			}
		}
		break
	case reflect.TypeOf(low.NodeReference[string]{}):
		if valueNode != nil {
			if utils.IsNodeStringValue(valueNode) {
				if field.CanSet() {
					nr := low.NodeReference[string]{
						Value:     valueNode.Value,
						ValueNode: valueNode,
						KeyNode:   keyNode,
					}
					field.Set(reflect.ValueOf(nr))
				}
			}
		}
		break
	case reflect.TypeOf(low.NodeReference[bool]{}):
		if valueNode != nil {
			if utils.IsNodeBoolValue(valueNode) {
				if field.CanSet() {
					bv, _ := strconv.ParseBool(valueNode.Value)
					nr := low.NodeReference[bool]{
						Value:     bv,
						ValueNode: valueNode,
						KeyNode:   keyNode,
					}
					field.Set(reflect.ValueOf(nr))
				}
			}
		}
		break
	case reflect.TypeOf(low.NodeReference[int]{}):
		if valueNode != nil {
			if utils.IsNodeIntValue(valueNode) {
				if field.CanSet() {
					fv, _ := strconv.Atoi(valueNode.Value)
					nr := low.NodeReference[int]{
						Value:     fv,
						ValueNode: valueNode,
						KeyNode:   keyNode,
					}
					field.Set(reflect.ValueOf(nr))
				}
			}
		}
		break
	case reflect.TypeOf(low.NodeReference[int64]{}):
		if valueNode != nil {
			if utils.IsNodeIntValue(valueNode) || utils.IsNodeFloatValue(valueNode) { //
				if field.CanSet() {
					fv, _ := strconv.ParseInt(valueNode.Value, 10, 64)
					nr := low.NodeReference[int64]{
						Value:     fv,
						ValueNode: valueNode,
						KeyNode:   keyNode,
					}
					field.Set(reflect.ValueOf(nr))
				}
			}
		}
		break
	case reflect.TypeOf(low.NodeReference[float32]{}):
		if valueNode != nil {
			if utils.IsNodeFloatValue(valueNode) {
				if field.CanSet() {
					fv, _ := strconv.ParseFloat(valueNode.Value, 32)
					nr := low.NodeReference[float32]{
						Value:     float32(fv),
						ValueNode: valueNode,
						KeyNode:   keyNode,
					}
					field.Set(reflect.ValueOf(nr))
				}
			}
		}
		break
	case reflect.TypeOf(low.NodeReference[float64]{}):
		if valueNode != nil {
			if utils.IsNodeFloatValue(valueNode) {
				if field.CanSet() {
					fv, _ := strconv.ParseFloat(valueNode.Value, 64)
					nr := low.NodeReference[float64]{
						Value:     fv,
						ValueNode: valueNode,
						KeyNode:   keyNode,
					}
					field.Set(reflect.ValueOf(nr))
				}
			}
		}
		break
	case reflect.TypeOf([]low.NodeReference[string]{}):
		if valueNode != nil {
			if utils.IsNodeArray(valueNode) {
				if field.CanSet() {
					var items []low.NodeReference[string]
					for _, sliceItem := range valueNode.Content {
						items = append(items, low.NodeReference[string]{
							Value:     sliceItem.Value,
							ValueNode: sliceItem,
							KeyNode:   valueNode,
						})
					}
					field.Set(reflect.ValueOf(items))
				}
			}
		}
		break
	case reflect.TypeOf([]low.NodeReference[float32]{}):
		if valueNode != nil {
			if utils.IsNodeArray(valueNode) {
				if field.CanSet() {
					var items []low.NodeReference[float32]
					for _, sliceItem := range valueNode.Content {
						fv, _ := strconv.ParseFloat(sliceItem.Value, 32)
						items = append(items, low.NodeReference[float32]{
							Value:     float32(fv),
							ValueNode: sliceItem,
							KeyNode:   valueNode,
						})
					}
					field.Set(reflect.ValueOf(items))
				}
			}
		}
		break
	case reflect.TypeOf([]low.NodeReference[float64]{}):
		if valueNode != nil {
			if utils.IsNodeArray(valueNode) {
				if field.CanSet() {
					var items []low.NodeReference[float64]
					for _, sliceItem := range valueNode.Content {
						fv, _ := strconv.ParseFloat(sliceItem.Value, 64)
						items = append(items, low.NodeReference[float64]{Value: fv, ValueNode: sliceItem})
					}
					field.Set(reflect.ValueOf(items))
				}
			}
		}
		break
	case reflect.TypeOf([]low.NodeReference[int]{}):
		if valueNode != nil {
			if utils.IsNodeArray(valueNode) {
				if field.CanSet() {
					var items []low.NodeReference[int]
					for _, sliceItem := range valueNode.Content {
						iv, _ := strconv.Atoi(sliceItem.Value)
						items = append(items, low.NodeReference[int]{
							Value:     iv,
							ValueNode: sliceItem,
							KeyNode:   valueNode,
						})
					}
					field.Set(reflect.ValueOf(items))
				}
			}
		}
		break
	case reflect.TypeOf([]low.NodeReference[int64]{}):
		if valueNode != nil {
			if utils.IsNodeArray(valueNode) {
				if field.CanSet() {
					var items []low.NodeReference[int64]
					for _, sliceItem := range valueNode.Content {
						iv, _ := strconv.ParseInt(sliceItem.Value, 10, 64)
						items = append(items, low.NodeReference[int64]{
							Value:     iv,
							ValueNode: sliceItem,
							KeyNode:   valueNode,
						})
					}
					field.Set(reflect.ValueOf(items))
				}
			}
		}
		break
	case reflect.TypeOf([]low.NodeReference[bool]{}):
		if valueNode != nil {
			if utils.IsNodeArray(valueNode) {
				if field.CanSet() {
					var items []low.NodeReference[bool]
					for _, sliceItem := range valueNode.Content {
						bv, _ := strconv.ParseBool(sliceItem.Value)
						items = append(items, low.NodeReference[bool]{
							Value:     bv,
							ValueNode: sliceItem,
							KeyNode:   valueNode,
						})
					}
					field.Set(reflect.ValueOf(items))
				}
			}
		}
		break
	default:
		// we want to ignore everything else.
		break
	}
	return nil
}

func BuildModelAsync(n *yaml.Node, model interface{}, lwg *sync.WaitGroup, errors *[]error) {
	if n != nil {
		err := BuildModel(n, model)
		if err != nil {
			*errors = append(*errors, err)
		}
	}
	lwg.Done()
}

func ExtractExtensions(root *yaml.Node) (map[low.NodeReference[string]]low.NodeReference[any], error) {
	extensions := utils.FindExtensionNodes(root.Content)
	extensionMap := make(map[low.NodeReference[string]]low.NodeReference[any])
	for _, ext := range extensions {
		if utils.IsNodeMap(ext.Value) {
			var v interface{}
			err := ext.Value.Decode(&v)
			if err != nil {
				return nil, err
			}
			extensionMap[low.NodeReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.NodeReference[any]{Value: v, KeyNode: ext.Key}
		}
		if utils.IsNodeStringValue(ext.Value) {
			extensionMap[low.NodeReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.NodeReference[any]{Value: ext.Value.Value, ValueNode: ext.Value}
		}
		if utils.IsNodeFloatValue(ext.Value) {
			fv, _ := strconv.ParseFloat(ext.Value.Value, 64)
			extensionMap[low.NodeReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.NodeReference[any]{Value: fv, ValueNode: ext.Value}
		}
		if utils.IsNodeIntValue(ext.Value) {
			iv, _ := strconv.ParseInt(ext.Value.Value, 10, 64)
			extensionMap[low.NodeReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.NodeReference[any]{Value: iv, ValueNode: ext.Value}
		}
		if utils.IsNodeBoolValue(ext.Value) {
			bv, _ := strconv.ParseBool(ext.Value.Value)
			extensionMap[low.NodeReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.NodeReference[any]{Value: bv, ValueNode: ext.Value}
		}
		if utils.IsNodeArray(ext.Value) {
			var v []interface{}
			err := ext.Value.Decode(&v)
			if err != nil {
				return nil, err
			}
			extensionMap[low.NodeReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.NodeReference[any]{Value: v, ValueNode: ext.Value}
		}
	}
	return extensionMap, nil
}

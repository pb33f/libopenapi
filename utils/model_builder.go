package utils

import (
    "fmt"
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
    "reflect"
    "strconv"
)

func BuildModel(node *yaml.Node, model interface{}) error {
    v := reflect.ValueOf(model).Elem()
    for i := 0; i < v.NumField(); i++ {

        fName := v.Type().Field(i).Name

        // we need to find a matching field in the YAML, the cases may be off, so take no chances.
        cases := []Case{PascalCase, CamelCase, ScreamingSnakeCase,
            SnakeCase, KebabCase, RegularCase}

        var vn, kn *yaml.Node
        for _, tryCase := range cases {
            kn, vn = FindKeyNode(ConvertCase(fName, tryCase), node.Content)
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
        case reflect.Struct, reflect.Slice, reflect.Map:
            err := SetField(field, vn, kn)
            if err != nil {
                return nil
            }
        default:
            fmt.Printf("Unsupported type: %v", v.Field(i).Kind())
            break
        }

    }

    return nil
}

func SetField(field reflect.Value, valueNode *yaml.Node, keyNode *yaml.Node) error {
    switch field.Type() {

    case reflect.TypeOf(map[string]low.ObjectReference{}):
        if valueNode != nil {
            if IsNodeMap(valueNode) {
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
                            Value: decoded,
                            Node:  sliceItem,
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
            if IsNodeMap(valueNode) {
                if field.CanSet() {
                    or := low.ObjectReference{Value: decoded, Node: valueNode}
                    field.Set(reflect.ValueOf(or))
                }
            }
        }
        break
    case reflect.TypeOf([]low.ObjectReference{}):
        if valueNode != nil {
            if IsNodeArray(valueNode) {
                if field.CanSet() {
                    var items []low.ObjectReference
                    for _, sliceItem := range valueNode.Content {
                        var decoded map[string]interface{}
                        err := sliceItem.Decode(&decoded)
                        if err != nil {
                            return err
                        }
                        items = append(items, low.ObjectReference{
                            Value: decoded,
                            Node:  sliceItem,
                        })
                    }
                    field.Set(reflect.ValueOf(items))
                }
            }
        }
        break
    case reflect.TypeOf(low.NodeReference[string]{}):
        if valueNode != nil {
            if IsNodeStringValue(valueNode) {
                if field.CanSet() {
                    nr := low.NodeReference[string]{Value: valueNode.Value, Node: valueNode}
                    field.Set(reflect.ValueOf(nr))
                }
            }
        }
        break
    case reflect.TypeOf(low.NodeReference[bool]{}):
        if valueNode != nil {
            if IsNodeBoolValue(valueNode) {
                if field.CanSet() {
                    bv, _ := strconv.ParseBool(valueNode.Value)
                    nr := low.NodeReference[bool]{Value: bv, Node: valueNode}
                    field.Set(reflect.ValueOf(nr))
                }
            }
        }
        break
    case reflect.TypeOf(low.NodeReference[int]{}):
        if valueNode != nil {
            if IsNodeIntValue(valueNode) {
                if field.CanSet() {
                    fv, _ := strconv.Atoi(valueNode.Value)
                    nr := low.NodeReference[int]{Value: fv, Node: valueNode}
                    field.Set(reflect.ValueOf(nr))
                }
            }
        }
        break
    case reflect.TypeOf(low.NodeReference[int64]{}):
        if valueNode != nil {
            if IsNodeIntValue(valueNode) {
                if field.CanSet() {
                    fv, _ := strconv.Atoi(valueNode.Value)
                    nr := low.NodeReference[int64]{Value: int64(fv), Node: valueNode}
                    field.Set(reflect.ValueOf(nr))
                }
            }
        }
        break
    case reflect.TypeOf(low.NodeReference[float32]{}):
        if valueNode != nil {
            if IsNodeFloatValue(valueNode) {
                if field.CanSet() {
                    fv, _ := strconv.ParseFloat(valueNode.Value, 32)
                    nr := low.NodeReference[float32]{Value: float32(fv), Node: valueNode}
                    field.Set(reflect.ValueOf(nr))
                }
            }
        }
        break
    case reflect.TypeOf(low.NodeReference[float64]{}):
        if valueNode != nil {
            if IsNodeFloatValue(valueNode) {
                if field.CanSet() {
                    fv, _ := strconv.ParseFloat(valueNode.Value, 64)
                    nr := low.NodeReference[float64]{Value: fv, Node: valueNode}
                    field.Set(reflect.ValueOf(nr))
                }
            }
        }
        break
    case reflect.TypeOf([]low.NodeReference[string]{}):
        if valueNode != nil {
            if IsNodeArray(valueNode) {
                if field.CanSet() {
                    var items []low.NodeReference[string]
                    for _, sliceItem := range valueNode.Content {
                        items = append(items, low.NodeReference[string]{Value: sliceItem.Value, Node: sliceItem})
                    }
                    field.Set(reflect.ValueOf(items))
                }
            }
        }
        break
    case reflect.TypeOf([]low.NodeReference[float32]{}):
        if valueNode != nil {
            if IsNodeArray(valueNode) {
                if field.CanSet() {
                    var items []low.NodeReference[float32]
                    for _, sliceItem := range valueNode.Content {
                        fv, _ := strconv.ParseFloat(sliceItem.Value, 32)
                        items = append(items, low.NodeReference[float32]{Value: float32(fv), Node: sliceItem})
                    }
                    field.Set(reflect.ValueOf(items))
                }
            }
        }
        break
    case reflect.TypeOf([]low.NodeReference[float64]{}):
        if valueNode != nil {
            if IsNodeArray(valueNode) {
                if field.CanSet() {
                    var items []low.NodeReference[float64]
                    for _, sliceItem := range valueNode.Content {
                        fv, _ := strconv.ParseFloat(sliceItem.Value, 64)
                        items = append(items, low.NodeReference[float64]{Value: fv, Node: sliceItem})
                    }
                    field.Set(reflect.ValueOf(items))
                }
            }
        }
        break
    case reflect.TypeOf([]low.NodeReference[int]{}):
        if valueNode != nil {
            if IsNodeArray(valueNode) {
                if field.CanSet() {
                    var items []low.NodeReference[int]
                    for _, sliceItem := range valueNode.Content {
                        iv, _ := strconv.Atoi(sliceItem.Value)
                        items = append(items, low.NodeReference[int]{Value: iv, Node: sliceItem})
                    }
                    field.Set(reflect.ValueOf(items))
                }
            }
        }
        break
    case reflect.TypeOf([]low.NodeReference[bool]{}):
        if valueNode != nil {
            if IsNodeArray(valueNode) {
                if field.CanSet() {
                    var items []low.NodeReference[bool]
                    for _, sliceItem := range valueNode.Content {
                        bv, _ := strconv.ParseBool(sliceItem.Value)
                        items = append(items, low.NodeReference[bool]{Value: bv, Node: sliceItem})
                    }
                    field.Set(reflect.ValueOf(items))
                }
            }
        }
        break
    default:
        m := field.Type()
        fmt.Printf("error, unknown type!!! %v", m)
        return fmt.Errorf("unknown type, cannot parse: %v", m)
    }
    return nil
}

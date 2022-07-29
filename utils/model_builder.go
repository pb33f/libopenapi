package utils

import (
    "fmt"
    "github.com/iancoleman/strcase"
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
    "reflect"
    "strconv"
)

func BuildModel(node *yaml.Node, model interface{}) error {
    v := reflect.ValueOf(model).Elem()
    for i := 0; i < v.NumField(); i++ {

        fName := v.Type().Field(i).Name
        fieldName := strcase.ToLowerCamel(fName)
        _, vn := FindKeyNode(fieldName, node.Content)
        field := v.FieldByName(fName)
        switch field.Kind() {
        case reflect.Struct:
            switch field.Type() {
            case reflect.TypeOf(low.NodeReference[string]{}):
                if vn != nil {
                    if IsNodeStringValue(vn) {
                        if field.CanSet() {
                            nr := low.NodeReference[string]{Value: vn.Value, Node: vn}
                            field.Set(reflect.ValueOf(nr))
                        }
                    }
                }
                break

            case reflect.TypeOf(low.NodeReference[bool]{}):
                if vn != nil {
                    if IsNodeBoolValue(vn) {
                        if field.CanSet() {
                            bv, _ := strconv.ParseBool(vn.Value)
                            nr := low.NodeReference[bool]{Value: bv, Node: vn}
                            field.Set(reflect.ValueOf(nr))
                        }
                    }
                }
            case reflect.TypeOf(low.NodeReference[int]{}):
                if vn != nil {
                    if IsNodeIntValue(vn) {
                        if field.CanSet() {
                            fv, _ := strconv.Atoi(vn.Value)
                            nr := low.NodeReference[int]{Value: fv, Node: vn}
                            field.Set(reflect.ValueOf(nr))
                        }
                    }
                }
            case reflect.TypeOf(low.NodeReference[int64]{}):
                if vn != nil {
                    if IsNodeIntValue(vn) {
                        if field.CanSet() {
                            fv, _ := strconv.Atoi(vn.Value)
                            nr := low.NodeReference[int64]{Value: int64(fv), Node: vn}
                            field.Set(reflect.ValueOf(nr))
                        }
                    }
                }
            case reflect.TypeOf(low.NodeReference[float32]{}):
                if vn != nil {
                    if IsNodeFloatValue(vn) {
                        if field.CanSet() {
                            fv, _ := strconv.ParseFloat(vn.Value, 32)
                            nr := low.NodeReference[float32]{Value: float32(fv), Node: vn}
                            field.Set(reflect.ValueOf(nr))
                        }
                    }
                }
            case reflect.TypeOf(low.NodeReference[float64]{}):
                if vn != nil {
                    if IsNodeFloatValue(vn) {
                        if field.CanSet() {
                            fv, _ := strconv.ParseFloat(vn.Value, 64)
                            nr := low.NodeReference[float64]{Value: fv, Node: vn}
                            field.Set(reflect.ValueOf(nr))
                        }
                    }
                }
            }
        default:
            fmt.Printf("Unsupported type: %v", v.Field(i).Kind())
            break
        }

    }

    return nil
}

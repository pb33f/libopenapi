package utils

import (
    "fmt"
    "github.com/iancoleman/strcase"
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
        switch v.Field(i).Kind() {
        case reflect.Int:
            if vn != nil {
                if IsNodeIntValue(vn) {
                    if field.CanSet() {
                        fv, _ := strconv.Atoi(vn.Value)
                        field.SetInt(int64(fv))
                    }
                }
            }
            break
        case reflect.String:
            if vn != nil {
                if IsNodeStringValue(vn) {
                    if field.CanSet() {
                        field.SetString(vn.Value)
                    }
                }
            }
            break
        case reflect.Bool:
            if vn != nil {
                if IsNodeBoolValue(vn) {
                    if field.CanSet() {
                        bv, _ := strconv.ParseBool(vn.Value)
                        field.SetBool(bv)
                    }
                }
            }
            break
        case reflect.Array:
            // TODO
            break
        case reflect.Float32, reflect.Float64:
            if vn != nil {
                if IsNodeFloatValue(vn) {
                    if field.CanSet() {
                        fv, _ := strconv.ParseFloat(vn.Value, 64)
                        field.SetFloat(fv)
                    }
                }
            }
            break
        case reflect.Struct:
            // TODO
            break

        default:
            fmt.Printf("Unsupported type: %v", v.Field(i).Kind())
            break
        }

    }

    return nil
}

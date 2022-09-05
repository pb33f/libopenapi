package datamodel

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

const (
	OAS2  = "oas2"
	OAS3  = "oas3"
	OAS31 = "oas3_1"
)

//go:embed schemas/oas3-schema.json
var OpenAPI3SchemaData string

//go:embed schemas/swagger2-schema.json
var OpenAPI2SchemaData string

var OAS3_1Format = []string{OAS31}
var OAS3Format = []string{OAS3}
var OAS3AllFormat = []string{OAS3, OAS31}
var OAS2Format = []string{OAS2}
var AllFormats = []string{OAS3, OAS31, OAS2}

// ExtractSpecInfo will look at a supplied OpenAPI specification, and return a *SpecInfo pointer, or an error
// if the spec cannot be parsed correctly.
func ExtractSpecInfo(spec []byte) (*SpecInfo, error) {

	var parsedSpec yaml.Node

	specVersion := &SpecInfo{}
	specVersion.JsonParsingChannel = make(chan bool)

	// set original bytes
	specVersion.SpecBytes = &spec

	runes := []rune(strings.TrimSpace(string(spec)))
	if len(runes) <= 0 {
		return specVersion, errors.New("there are no runes in the spec")
	}

	if runes[0] == '{' && runes[len(runes)-1] == '}' {
		specVersion.SpecFileType = "json"
	} else {
		specVersion.SpecFileType = "yaml"
	}

	err := yaml.Unmarshal(spec, &parsedSpec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse specification: %s", err.Error())
	}

	specVersion.RootNode = &parsedSpec

	_, openAPI3 := utils.FindKeyNode(utils.OpenApi3, parsedSpec.Content)
	_, openAPI2 := utils.FindKeyNode(utils.OpenApi2, parsedSpec.Content)
	_, asyncAPI := utils.FindKeyNode(utils.AsyncApi, parsedSpec.Content)

	parseJSON := func(bytes []byte, spec *SpecInfo) {
		var jsonSpec map[string]interface{}

		// no point in worrying about errors here, extract JSON friendly format.
		// run in a separate thread, don't block.

		if spec.SpecType == utils.OpenApi3 {
			spec.APISchema = OpenAPI3SchemaData
		}
		if spec.SpecType == utils.OpenApi2 {
			spec.APISchema = OpenAPI2SchemaData
		}

		if utils.IsYAML(string(bytes)) {
			yaml.Unmarshal(bytes, &jsonSpec)
			jsonData, _ := json.Marshal(jsonSpec)
			spec.SpecJSONBytes = &jsonData
			spec.SpecJSON = &jsonSpec
		} else {
			json.Unmarshal(bytes, &jsonSpec)
			spec.SpecJSONBytes = &bytes
			spec.SpecJSON = &jsonSpec
		}
		spec.JsonParsingChannel <- true
		close(spec.JsonParsingChannel)
	}
	// check for specific keys
	if openAPI3 != nil {
		specVersion.SpecType = utils.OpenApi3
		version, majorVersion, versionError := parseVersionTypeData(openAPI3.Value)
		if versionError != nil {
			return nil, versionError
		}

		// parse JSON
		go parseJSON(spec, specVersion)

		// double check for the right version, people mix this up.
		if majorVersion < 3 {
			specVersion.Error = errors.New("spec is defined as an openapi spec, but is using a swagger (2.0), or unknown version")
			return specVersion, specVersion.Error
		}
		specVersion.Version = version
		specVersion.SpecFormat = OAS3
	}
	if openAPI2 != nil {
		specVersion.SpecType = utils.OpenApi2
		version, majorVersion, versionError := parseVersionTypeData(openAPI2.Value)
		if versionError != nil {
			return nil, versionError
		}

		// parse JSON
		go parseJSON(spec, specVersion)

		// I am not certain this edge-case is very frequent, but let's make sure we handle it anyway.
		if majorVersion > 2 {
			specVersion.Error = errors.New("spec is defined as a swagger (openapi 2.0) spec, but is an openapi 3 or unknown version")
			return specVersion, specVersion.Error
		}
		specVersion.Version = version
		specVersion.SpecFormat = OAS2
	}
	if asyncAPI != nil {
		specVersion.SpecType = utils.AsyncApi
		version, majorVersion, versionErr := parseVersionTypeData(asyncAPI.Value)
		if versionErr != nil {
			return nil, versionErr
		}

		// parse JSON
		go parseJSON(spec, specVersion)

		// so far there is only 2 as a major release of AsyncAPI
		if majorVersion > 2 {
			specVersion.Error = errors.New("spec is defined as asyncapi, but has a major version that is invalid")
			return specVersion, specVersion.Error
		}
		specVersion.Version = version
		// TODO: format for AsyncAPI.

	}

	if specVersion.SpecType == "" {

		// parse JSON
		go parseJSON(spec, specVersion)

		specVersion.Error = errors.New("spec type not supported by vacuum, sorry")
		return specVersion, specVersion.Error
	}

	return specVersion, nil
}

func parseVersionTypeData(d interface{}) (string, int, error) {
	r := []rune(strings.TrimSpace(fmt.Sprintf("%v", d)))
	if len(r) <= 0 {
		return "", 0, fmt.Errorf("unable to extract version from: %v", d)
	}
	return string(r), int(r[0]) - '0', nil
}

// AreValuesCorrectlyTyped will look through an array of unknown values and check they match
// against the supplied type as a string. The return value is empty if everything is OK, or it
// contains failures in the form of a value as a key and a message as to why it's not valid
func AreValuesCorrectlyTyped(valType string, values interface{}) map[string]string {
	var arr []interface{}
	if _, ok := values.([]interface{}); !ok {
		return nil
	}
	arr = values.([]interface{})

	results := make(map[string]string)
	for _, v := range arr {
		switch v.(type) {
		case string:
			if valType != "string" {
				results[v.(string)] = fmt.Sprintf("enum value '%v' is a "+
					"string, but it's defined as a '%v'", v, valType)
			}
		case int64:
			if valType != "integer" && valType != "number" {
				results[fmt.Sprintf("%v", v)] = fmt.Sprintf("enum value '%v' is a "+
					"integer, but it's defined as a '%v'", v, valType)
			}
		case int:
			if valType != "integer" && valType != "number" {
				results[fmt.Sprintf("%v", v)] = fmt.Sprintf("enum value '%v' is a "+
					"integer, but it's defined as a '%v'", v, valType)
			}
		case float64:
			if valType != "number" {
				results[fmt.Sprintf("%v", v)] = fmt.Sprintf("enum value '%v' is a "+
					"number, but it's defined as a '%v'", v, valType)
			}
		case bool:
			if valType != "boolean" {
				results[fmt.Sprintf("%v", v)] = fmt.Sprintf("enum value '%v' is a "+
					"boolean, but it's defined as a '%v'", v, valType)
			}
		}
	}
	return results
}

// CheckEnumForDuplicates will check an array of nodes to check if there are any duplicates.
func CheckEnumForDuplicates(seq []*yaml.Node) []*yaml.Node {
	var res []*yaml.Node
	seen := make(map[string]*yaml.Node)

	for _, enum := range seq {
		if seen[enum.Value] != nil {
			res = append(res, enum)
			continue
		}
		seen[enum.Value] = enum
	}
	return res
}

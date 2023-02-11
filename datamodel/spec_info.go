// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

const (
	JSONFileType = "json"
	YAMLFileType = "yaml"
)

// SpecInfo represents a 'ready-to-process' OpenAPI Document. The RootNode is the most important property
// used by the library, this contains the top of the document tree that every single low model is based off.
type SpecInfo struct {
	SpecType           string                  `json:"type"`
	Version            string                  `json:"version"`
	SpecFormat         string                  `json:"format"`
	SpecFileType       string                  `json:"fileType"`
	SpecBytes          *[]byte                 `json:"bytes"` // the original byte array
	RootNode           *yaml.Node              `json:"-"`     // reference to the root node of the spec.
	SpecJSONBytes      *[]byte                 `json:"-"`     // original bytes converted to JSON
	SpecJSON           *map[string]interface{} `json:"-"`     // standard JSON map of original bytes
	Error              error                   `json:"-"`     // something go wrong?
	APISchema          string                  `json:"-"`     // API Schema for supplied spec type (2 or 3)
	Generated          time.Time               `json:"-"`
	JsonParsingChannel chan bool               `json:"-"`
}

// GetJSONParsingChannel returns a channel that will close once async JSON parsing is completed.
// This is really useful if your application wants to analyze the JSON via SpecJSON. the library will
// return *SpecInfo BEFORE the JSON is done parsing, so things are as fast as possible.
//
// If you want to know when parsing is done, listen on the channel for a bool.
func (si SpecInfo) GetJSONParsingChannel() chan bool {
	return si.JsonParsingChannel
}

// ExtractSpecInfo accepts an OpenAPI/Swagger specification that has been read into a byte array
// and will return a SpecInfo pointer, which contains details on the version and an un-marshaled
// *yaml.Node root node tree. The root node tree is what's used by the library when building out models.
//
// If the spec cannot be parsed correctly then an error will be returned, otherwise the error is nil.
func ExtractSpecInfo(spec []byte) (*SpecInfo, error) {

	var parsedSpec yaml.Node

	specVersion := &SpecInfo{}
	specVersion.JsonParsingChannel = make(chan bool)

	// set original bytes
	specVersion.SpecBytes = &spec

	runes := []rune(strings.TrimSpace(string(spec)))
	if len(runes) <= 0 {
		return specVersion, errors.New("there is nothing in the spec, it's empty - so there is nothing to be done")
	}

	if runes[0] == '{' && runes[len(runes)-1] == '}' {
		specVersion.SpecFileType = JSONFileType
	} else {
		specVersion.SpecFileType = YAMLFileType
	}

	err := yaml.Unmarshal(spec, &parsedSpec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse specification: %s", err.Error())
	}

	specVersion.RootNode = &parsedSpec

	_, openAPI3 := utils.FindKeyNode(utils.OpenApi3, parsedSpec.Content)
	_, openAPI2 := utils.FindKeyNode(utils.OpenApi2, parsedSpec.Content)
	_, asyncAPI := utils.FindKeyNode(utils.AsyncApi, parsedSpec.Content)

	parseJSON := func(bytes []byte, spec *SpecInfo, parsedNode *yaml.Node) {
		var jsonSpec map[string]interface{}

		if spec.SpecType == utils.OpenApi3 {
			switch spec.Version {
			case "3.1.0", "3.1":
				spec.APISchema = OpenAPI31SchemaData
			default:
				spec.APISchema = OpenAPI3SchemaData
			}
		}
		if spec.SpecType == utils.OpenApi2 {
			spec.APISchema = OpenAPI2SchemaData
		}

		if utils.IsYAML(string(bytes)) {
			_ = parsedNode.Decode(&jsonSpec)
			b, _ := json.Marshal(&jsonSpec)
			spec.SpecJSONBytes = &b
			spec.SpecJSON = &jsonSpec
		} else {
			_ = json.Unmarshal(bytes, &jsonSpec)
			spec.SpecJSONBytes = &bytes
			spec.SpecJSON = &jsonSpec
		}
		close(spec.JsonParsingChannel) // this needs removing at some point
	}

	// check for specific keys
	if openAPI3 != nil {
		version, majorVersion, versionError := parseVersionTypeData(openAPI3.Value)
		if versionError != nil {
			return nil, versionError
		}

		specVersion.SpecType = utils.OpenApi3
		specVersion.Version = version
		specVersion.SpecFormat = OAS3

		// parse JSON
		parseJSON(spec, specVersion, &parsedSpec)

		// double check for the right version, people mix this up.
		if majorVersion < 3 {
			specVersion.Error = errors.New("spec is defined as an openapi spec, but is using a swagger (2.0), or unknown version")
			return specVersion, specVersion.Error
		}
	}

	if openAPI2 != nil {
		version, majorVersion, versionError := parseVersionTypeData(openAPI2.Value)
		if versionError != nil {
			return nil, versionError
		}

		specVersion.SpecType = utils.OpenApi2
		specVersion.Version = version
		specVersion.SpecFormat = OAS2

		// parse JSON
		parseJSON(spec, specVersion, &parsedSpec)

		// I am not certain this edge-case is very frequent, but let's make sure we handle it anyway.
		if majorVersion > 2 {
			specVersion.Error = errors.New("spec is defined as a swagger (openapi 2.0) spec, but is an openapi 3 or unknown version")
			return specVersion, specVersion.Error
		}
	}
	if asyncAPI != nil {
		version, majorVersion, versionErr := parseVersionTypeData(asyncAPI.Value)
		if versionErr != nil {
			return nil, versionErr
		}

		specVersion.SpecType = utils.AsyncApi
		specVersion.Version = version
		// TODO: format for AsyncAPI.

		// parse JSON
		parseJSON(spec, specVersion, &parsedSpec)

		// so far there is only 2 as a major release of AsyncAPI
		if majorVersion > 2 {
			specVersion.Error = errors.New("spec is defined as asyncapi, but has a major version that is invalid")
			return specVersion, specVersion.Error
		}
	}

	if specVersion.SpecType == "" {
		// parse JSON
		parseJSON(spec, specVersion, &parsedSpec)
		specVersion.Error = errors.New("spec type not supported by libopenapi, sorry")
		return specVersion, specVersion.Error
	}

	return specVersion, nil
}

// extract version number from specification
func parseVersionTypeData(d interface{}) (string, int, error) {
	r := []rune(strings.TrimSpace(fmt.Sprintf("%v", d)))
	if len(r) <= 0 {
		return "", 0, fmt.Errorf("unable to extract version from: %v", d)
	}
	return string(r), int(r[0]) - '0', nil
}

// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package sdk

import (
	"errors"
	"fmt"
	"strings"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	highv3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

// Prepare builds the effective operation contract consumed by language emitters.
func Prepare(document *highv3.Document, options PrepareOptions) (*Contract, error) {
	if document == nil {
		return nil, errors.New("sdk: OpenAPI document is required")
	}
	contract := &Contract{Schemas: orderedmap.New[string, *highbase.SchemaProxy]()}
	for _, server := range document.Servers {
		if server != nil && strings.TrimSpace(server.URL) != "" {
			resolved, err := resolveServerURL(server)
			if err != nil {
				return nil, fmt.Errorf("sdk: prepare document server %q: %w", server.URL, err)
			}
			contract.Servers = append(contract.Servers, resolved)
		}
	}
	if document.Components != nil {
		if document.Components.Schemas != nil {
			contract.Schemas = document.Components.Schemas
		}
		if document.Components.SecuritySchemes != nil {
			for name, scheme := range document.Components.SecuritySchemes.FromOldest() {
				if scheme == nil {
					continue
				}
				contract.SecuritySchemes = append(contract.SecuritySchemes, &SecurityScheme{
					Name: name, Type: scheme.Type, In: scheme.In, ParameterName: scheme.Name,
					Scheme: scheme.Scheme, OpenIDConnectURL: scheme.OpenIdConnectUrl,
				})
			}
		}
	}
	selected := make(map[string]struct{}, len(options.Operations))
	for _, operationID := range options.Operations {
		operationID = strings.TrimSpace(operationID)
		if operationID == "" {
			return nil, errors.New("sdk: selected operationId cannot be empty")
		}
		selected[operationID] = struct{}{}
	}
	seen := make(map[string]string)
	if document.Paths == nil || document.Paths.PathItems == nil {
		if len(selected) > 0 {
			return nil, errors.New("sdk: OpenAPI document has no paths")
		}
		return contract, nil
	}
	for path, item := range document.Paths.PathItems.FromOldest() {
		if item == nil {
			continue
		}
		for method, operation := range item.GetOperations().FromOldest() {
			if operation == nil || strings.TrimSpace(operation.OperationId) == "" {
				continue
			}
			method = strings.ToUpper(method)
			operationID := operation.OperationId
			if len(selected) > 0 {
				if _, ok := selected[operationID]; !ok {
					continue
				}
			}
			if prior, ok := seen[operationID]; ok {
				return nil, fmt.Errorf("sdk: duplicate operationId %q at %s and %s %s", operationID, prior, method, path)
			}
			seen[operationID] = method + " " + path
			prepared, err := prepareOperation(document, contract.Servers, item, operation, method, path, options.Names[operationID])
			if err != nil {
				return nil, fmt.Errorf("sdk: prepare %s %s (%s): %w", method, path, operationID, err)
			}
			contract.Operations = append(contract.Operations, prepared)
		}
	}
	for _, operationID := range options.Operations {
		if _, ok := seen[operationID]; !ok {
			return nil, fmt.Errorf("sdk: selected operationId %q was not found", operationID)
		}
	}
	return contract, nil
}

func prepareOperation(document *highv3.Document, documentServers []string, item *highv3.PathItem, operation *highv3.Operation, method, path string, override OperationName) (*Operation, error) {
	resource := "Default"
	if len(operation.Tags) > 0 && strings.TrimSpace(operation.Tags[0]) != "" {
		resource = operation.Tags[0]
	}
	if strings.TrimSpace(override.Resource) != "" {
		resource = override.Resource
	}
	name := methodSeed(operation.OperationId, resource)
	if strings.TrimSpace(override.Method) != "" {
		name = override.Method
	}
	prepared := &Operation{
		ID: operation.OperationId, Method: method, Path: path, Resource: resource, Name: name,
		Summary: operation.Summary, Description: operation.Description,
		Parameters:  mergeParameters(item.Parameters, operation.Parameters),
		RequestBody: prepareRequestBody(operation.RequestBody),
		Responses:   prepareResponses(operation.Responses),
	}
	servers := operation.Servers
	if len(servers) == 0 {
		servers = item.Servers
	}
	if len(servers) == 0 {
		prepared.Servers = documentServers
	} else {
		for _, server := range servers {
			if server != nil && strings.TrimSpace(server.URL) != "" {
				resolved, err := resolveServerURL(server)
				if err != nil {
					return nil, fmt.Errorf("prepare server %q: %w", server.URL, err)
				}
				prepared.Servers = append(prepared.Servers, resolved)
			}
		}
	}
	if len(prepared.Servers) == 0 {
		prepared.Servers = []string{"/"}
	}
	security := operation.Security
	if security == nil {
		security = document.Security
	}
	prepared.Security = prepareSecurity(security)
	return prepared, nil
}

func resolveServerURL(server *highv3.Server) (string, error) {
	resolved := server.URL
	if server.Variables != nil {
		for name, variable := range server.Variables.FromOldest() {
			if variable == nil {
				return "", fmt.Errorf("server variable %q has no definition", name)
			}
			placeholder := "{" + name + "}"
			if strings.Contains(resolved, placeholder) {
				resolved = strings.ReplaceAll(resolved, placeholder, variable.Default)
			}
		}
	}
	if strings.ContainsAny(resolved, "{}") {
		return "", errors.New("server URL contains an undeclared or malformed variable placeholder")
	}
	return resolved, nil
}

func mergeParameters(pathParameters, operationParameters []*highv3.Parameter) []*Parameter {
	positions := make(map[string]int)
	result := make([]*Parameter, 0, len(pathParameters)+len(operationParameters))
	appendOrReplace := func(parameter *highv3.Parameter) {
		if parameter == nil {
			return
		}
		key := parameter.In + "\x00" + parameter.Name
		prepared := prepareParameter(parameter)
		if position, ok := positions[key]; ok {
			result[position] = prepared
			return
		}
		positions[key] = len(result)
		result = append(result, prepared)
	}
	for _, parameter := range pathParameters {
		appendOrReplace(parameter)
	}
	for _, parameter := range operationParameters {
		appendOrReplace(parameter)
	}
	return result
}

func prepareParameter(parameter *highv3.Parameter) *Parameter {
	required := parameter.In == "path" || parameter.Required != nil && *parameter.Required
	return &Parameter{
		Name: parameter.Name, In: parameter.In, Description: parameter.Description,
		Required: required, Style: parameter.EffectiveStyle(), Explode: parameter.EffectiveExplode(),
		AllowReserved: parameter.AllowReserved, Schema: parameter.Schema,
	}
}

func prepareRequestBody(body *highv3.RequestBody) *RequestBody {
	if body == nil {
		return nil
	}
	required := body.Required != nil && *body.Required
	return &RequestBody{Required: required, Description: body.Description, Content: body.Content}
}

func prepareResponses(responses *highv3.Responses) []*Response {
	if responses == nil {
		return nil
	}
	result := make([]*Response, 0)
	if responses.Codes != nil {
		for status, response := range responses.Codes.FromOldest() {
			if response != nil {
				result = append(result, &Response{Status: status, Description: response.Description, Content: response.Content})
			}
		}
	}
	if responses.Default != nil {
		result = append(result, &Response{Status: "default", Description: responses.Default.Description, Content: responses.Default.Content})
	}
	return result
}

func prepareSecurity(requirements []*highbase.SecurityRequirement) []*SecurityRequirement {
	if requirements == nil {
		return nil
	}
	result := make([]*SecurityRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		prepared := &SecurityRequirement{}
		if requirement != nil && requirement.Requirements != nil {
			for name, scopes := range requirement.Requirements.FromOldest() {
				prepared.Schemes = append(prepared.Schemes, &RequiredSecurityScheme{Name: name, Scopes: append([]string(nil), scopes...)})
			}
		}
		result = append(result, prepared)
	}
	return result
}

func methodSeed(operationID, resource string) string {
	if len(resource) > 0 && len(operationID) > len(resource) && strings.EqualFold(operationID[len(operationID)-len(resource):], resource) {
		return operationID[:len(operationID)-len(resource)]
	}
	return operationID
}

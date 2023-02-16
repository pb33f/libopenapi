// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
    "fmt"
    "github.com/pb33f/libopenapi/utils"
    "gopkg.in/yaml.v3"
    "sync"
)

func (index *SpecIndex) extractDefinitionsAndSchemas(schemasNode *yaml.Node, pathPrefix string) {
    var name string
    for i, schema := range schemasNode.Content {
        if i%2 == 0 {
            name = schema.Value
            continue
        }

        def := fmt.Sprintf("%s%s", pathPrefix, name)
        ref := &Reference{
            Definition:            def,
            Name:                  name,
            Node:                  schema,
            Path:                  fmt.Sprintf("$.components.schemas.%s", name),
            ParentNode:            schemasNode,
            RequiredRefProperties: index.extractDefinitionRequiredRefProperties(schemasNode, map[string][]string{}),
        }
        index.allComponentSchemaDefinitions[def] = ref
    }
}

// extractDefinitionRequiredRefProperties goes through the direct properties of a schema and extracts the map of required definitions from within it
func (index *SpecIndex) extractDefinitionRequiredRefProperties(schemaNode *yaml.Node, reqRefProps map[string][]string) map[string][]string {
    if schemaNode == nil {
        return reqRefProps
    }

    // If the node we're looking at is a direct ref to another model without any properties, mark it as required, but still continue to look for required properties
    isRef, _, defPath := utils.IsNodeRefValue(schemaNode)
    if isRef {
        if _, ok := reqRefProps[defPath]; !ok {
            reqRefProps[defPath] = []string{}
        }
    }

    // Check for a required parameters list, and return if none exists, as any properties will be optional
    _, requiredSeqNode := utils.FindKeyNodeTop("required", schemaNode.Content)
    if requiredSeqNode == nil {
        return reqRefProps
    }

    _, propertiesMapNode := utils.FindKeyNodeTop("properties", schemaNode.Content)
    if propertiesMapNode == nil {
        // TODO: Log a warning on the resolver, because if you have required properties, but no actual properties, something is wrong
        return reqRefProps
    }

    name := ""
    for i, param := range propertiesMapNode.Content {
        if i%2 == 0 {
            name = param.Value
            continue
        }

        // Check to see if the current property is directly embedded within the current schema, and handle its properties if so
        _, paramPropertiesMapNode := utils.FindKeyNodeTop("properties", param.Content)
        if paramPropertiesMapNode != nil {
            reqRefProps = index.extractDefinitionRequiredRefProperties(param, reqRefProps)
        }

        // Check to see if the current property is polymorphic, and dive into that model if so
        for _, key := range []string{"allOf", "oneOf", "anyOf"} {
            _, ofNode := utils.FindKeyNodeTop(key, param.Content)
            if ofNode != nil {
                for _, ofNodeItem := range ofNode.Content {
                    reqRefProps = index.extractRequiredReferenceProperties(ofNodeItem, name, reqRefProps)
                }
            }
        }
    }

    // Run through each of the required properties and extract _their_ required references
    for _, requiredPropertyNode := range requiredSeqNode.Content {
        _, requiredPropDefNode := utils.FindKeyNodeTop(requiredPropertyNode.Value, propertiesMapNode.Content)
        if requiredPropDefNode == nil {
            continue
        }

        reqRefProps = index.extractRequiredReferenceProperties(requiredPropDefNode, requiredPropertyNode.Value, reqRefProps)
    }

    return reqRefProps
}

// extractRequiredReferenceProperties returns a map of definition names to the property or properties which reference it within a node
func (index *SpecIndex) extractRequiredReferenceProperties(requiredPropDefNode *yaml.Node, propName string, reqRefProps map[string][]string) map[string][]string {
    isRef, _, defPath := utils.IsNodeRefValue(requiredPropDefNode)
    if !isRef {
        _, defItems := utils.FindKeyNodeTop("items", requiredPropDefNode.Content)
        if defItems != nil {
            isRef, _, defPath = utils.IsNodeRefValue(defItems)
        }
    }

    if /* still */ !isRef {
        return reqRefProps
    }

    if _, ok := reqRefProps[defPath]; !ok {
        reqRefProps[defPath] = []string{}
    }
    reqRefProps[defPath] = append(reqRefProps[defPath], propName)

    return reqRefProps
}

func (index *SpecIndex) extractComponentParameters(paramsNode *yaml.Node, pathPrefix string) {
    var name string
    for i, param := range paramsNode.Content {
        if i%2 == 0 {
            name = param.Value
            continue
        }
        def := fmt.Sprintf("%s%s", pathPrefix, name)
        ref := &Reference{
            Definition: def,
            Name:       name,
            Node:       param,
        }
        index.allParameters[def] = ref
    }
}

func (index *SpecIndex) extractComponentRequestBodies(requestBodiesNode *yaml.Node, pathPrefix string) {
    var name string
    for i, reqBod := range requestBodiesNode.Content {
        if i%2 == 0 {
            name = reqBod.Value
            continue
        }
        def := fmt.Sprintf("%s%s", pathPrefix, name)
        ref := &Reference{
            Definition: def,
            Name:       name,
            Node:       reqBod,
        }
        index.allRequestBodies[def] = ref
    }
}

func (index *SpecIndex) extractComponentResponses(responsesNode *yaml.Node, pathPrefix string) {
    var name string
    for i, response := range responsesNode.Content {
        if i%2 == 0 {
            name = response.Value
            continue
        }
        def := fmt.Sprintf("%s%s", pathPrefix, name)
        ref := &Reference{
            Definition: def,
            Name:       name,
            Node:       response,
        }
        index.allResponses[def] = ref
    }
}

func (index *SpecIndex) extractComponentHeaders(headersNode *yaml.Node, pathPrefix string) {
    var name string
    for i, header := range headersNode.Content {
        if i%2 == 0 {
            name = header.Value
            continue
        }
        def := fmt.Sprintf("%s%s", pathPrefix, name)
        ref := &Reference{
            Definition: def,
            Name:       name,
            Node:       header,
        }
        index.allHeaders[def] = ref
    }
}

func (index *SpecIndex) extractComponentCallbacks(callbacksNode *yaml.Node, pathPrefix string) {
    var name string
    for i, callback := range callbacksNode.Content {
        if i%2 == 0 {
            name = callback.Value
            continue
        }
        def := fmt.Sprintf("%s%s", pathPrefix, name)
        ref := &Reference{
            Definition: def,
            Name:       name,
            Node:       callback,
        }
        index.allCallbacks[def] = ref
    }
}

func (index *SpecIndex) extractComponentLinks(linksNode *yaml.Node, pathPrefix string) {
    var name string
    for i, link := range linksNode.Content {
        if i%2 == 0 {
            name = link.Value
            continue
        }
        def := fmt.Sprintf("%s%s", pathPrefix, name)
        ref := &Reference{
            Definition: def,
            Name:       name,
            Node:       link,
        }
        index.allLinks[def] = ref
    }
}

func (index *SpecIndex) extractComponentExamples(examplesNode *yaml.Node, pathPrefix string) {
    var name string
    for i, example := range examplesNode.Content {
        if i%2 == 0 {
            name = example.Value
            continue
        }
        def := fmt.Sprintf("%s%s", pathPrefix, name)
        ref := &Reference{
            Definition: def,
            Name:       name,
            Node:       example,
        }
        index.allExamples[def] = ref
    }
}

func (index *SpecIndex) extractComponentSecuritySchemes(securitySchemesNode *yaml.Node, pathPrefix string) {
    var name string
    for i, secScheme := range securitySchemesNode.Content {
        if i%2 == 0 {
            name = secScheme.Value
            continue
        }
        def := fmt.Sprintf("%s%s", pathPrefix, name)
        ref := &Reference{
            Definition: def,
            Name:       name,
            Node:       secScheme,
            ParentNode: securitySchemesNode,
            Path:       fmt.Sprintf("$.components.securitySchemes.%s", name),
        }
        index.allSecuritySchemes[def] = ref
    }
}

func (index *SpecIndex) countUniqueInlineDuplicates() int {
    if index.componentsInlineParamUniqueCount > 0 {
        return index.componentsInlineParamUniqueCount
    }
    unique := 0
    for _, p := range index.paramInlineDuplicates {
        if len(p) == 1 {
            unique++
        }
    }
    index.componentsInlineParamUniqueCount = unique
    return unique
}

func (index *SpecIndex) scanOperationParams(params []*yaml.Node, pathItemNode *yaml.Node, method string) {
    for i, param := range params {
        // param is ref
        if len(param.Content) > 0 && param.Content[0].Value == "$ref" {

            paramRefName := param.Content[1].Value
            paramRef := index.allMappedRefs[paramRefName]

            if index.paramOpRefs[pathItemNode.Value] == nil {
                index.paramOpRefs[pathItemNode.Value] = make(map[string]map[string]*Reference)
                index.paramOpRefs[pathItemNode.Value][method] = make(map[string]*Reference)

            }
            // if we know the path, but it's a new method
            if index.paramOpRefs[pathItemNode.Value][method] == nil {
                index.paramOpRefs[pathItemNode.Value][method] = make(map[string]*Reference)
            }

            // if this is a duplicate, add an error and ignore it
            if index.paramOpRefs[pathItemNode.Value][method][paramRefName] != nil {
                path := fmt.Sprintf("$.paths.%s.%s.parameters[%d]", pathItemNode.Value, method, i)
                if method == "top" {
                    path = fmt.Sprintf("$.paths.%s.parameters[%d]", pathItemNode.Value, i)
                }

                index.operationParamErrors = append(index.operationParamErrors, &IndexingError{
                    Err: fmt.Errorf("the `%s` operation parameter at path `%s`, "+
                        "index %d has a duplicate ref `%s`", method, pathItemNode.Value, i, paramRefName),
                    Node: param,
                    Path: path,
                })
            } else {
                index.paramOpRefs[pathItemNode.Value][method][paramRefName] = paramRef
            }

            continue

        } else {

            // param is inline.
            _, vn := utils.FindKeyNode("name", param.Content)

            path := fmt.Sprintf("$.paths.%s.%s.parameters[%d]", pathItemNode.Value, method, i)
            if method == "top" {
                path = fmt.Sprintf("$.paths.%s.parameters[%d]", pathItemNode.Value, i)
            }

            if vn == nil {
                index.operationParamErrors = append(index.operationParamErrors, &IndexingError{
                    Err: fmt.Errorf("the '%s' operation parameter at path '%s', index %d has no 'name' value",
                        method, pathItemNode.Value, i),
                    Node: param,
                    Path: path,
                })
                continue
            }

            ref := &Reference{
                Definition: vn.Value,
                Name:       vn.Value,
                Node:       param,
                Path:       path,
            }
            if index.paramOpRefs[pathItemNode.Value] == nil {
                index.paramOpRefs[pathItemNode.Value] = make(map[string]map[string]*Reference)
                index.paramOpRefs[pathItemNode.Value][method] = make(map[string]*Reference)
            }

            // if we know the path but this is a new method.
            if index.paramOpRefs[pathItemNode.Value][method] == nil {
                index.paramOpRefs[pathItemNode.Value][method] = make(map[string]*Reference)
            }

            // if this is a duplicate, add an error and ignore it
            if index.paramOpRefs[pathItemNode.Value][method][ref.Name] != nil {
                path := fmt.Sprintf("$.paths.%s.%s.parameters[%d]", pathItemNode.Value, method, i)
                if method == "top" {
                    path = fmt.Sprintf("$.paths.%s.parameters[%d]", pathItemNode.Value, i)
                }

                index.operationParamErrors = append(index.operationParamErrors, &IndexingError{
                    Err: fmt.Errorf("the `%s` operation parameter at path `%s`, "+
                        "index %d has a duplicate name `%s`", method, pathItemNode.Value, i, vn.Value),
                    Node: param,
                    Path: path,
                })
            } else {
                index.paramOpRefs[pathItemNode.Value][method][ref.Name] = ref
            }
            continue
        }
    }
}

func runIndexFunction(funcs []func() int, wg *sync.WaitGroup) {
    for _, cFunc := range funcs {
        go func(wg *sync.WaitGroup, cf func() int) {
            cf()
            wg.Done()
        }(wg, cFunc)
    }
}

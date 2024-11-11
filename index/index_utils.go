// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"strings"
	"sync"
)

func isHttpMethod(val string) bool {
	switch strings.ToLower(val) {
	case methodTypes[0]:
		return true
	case methodTypes[1]:
		return true
	case methodTypes[2]:
		return true
	case methodTypes[3]:
		return true
	case methodTypes[4]:
		return true
	case methodTypes[5]:
		return true
	case methodTypes[6]:
		return true
	}
	return false
}

func boostrapIndexCollections(index *SpecIndex) {
	index.allRefs = make(map[string]*ReferenceNode)
	index.allMappedRefs = make(map[string]*ReferenceNode)
	index.refsByLine = make(map[string]map[int]bool)
	index.linesWithRefs = make(map[int]bool)
	index.pathRefs = make(map[string]map[string]*ReferenceNode)
	index.paramOpRefs = make(map[string]map[string]map[string][]*ReferenceNode)
	index.operationTagsRefs = make(map[string]map[string][]*ReferenceNode)
	index.operationDescriptionRefs = make(map[string]map[string]*ReferenceNode)
	index.operationSummaryRefs = make(map[string]map[string]*ReferenceNode)
	index.paramCompRefs = make(map[string]*ReferenceNode)
	index.paramAllRefs = make(map[string]*ReferenceNode)
	index.paramInlineDuplicateNames = make(map[string][]*ReferenceNode)
	index.globalTagRefs = make(map[string]*ReferenceNode)
	index.securitySchemeRefs = make(map[string]*ReferenceNode)
	index.requestBodiesRefs = make(map[string]*ReferenceNode)
	index.responsesRefs = make(map[string]*ReferenceNode)
	index.headersRefs = make(map[string]*ReferenceNode)
	index.examplesRefs = make(map[string]*ReferenceNode)
	index.callbacksRefs = make(map[string]map[string][]*ReferenceNode)
	index.linksRefs = make(map[string]map[string][]*ReferenceNode)
	index.callbackRefs = make(map[string]*ReferenceNode)
	index.externalSpecIndex = make(map[string]*SpecIndex)
	index.allComponentSchemaDefinitions = &sync.Map{}
	index.allParameters = make(map[string]*ReferenceNode)
	index.allSecuritySchemes = make(map[string]*ReferenceNode)
	index.allRequestBodies = make(map[string]*ReferenceNode)
	index.allResponses = make(map[string]*ReferenceNode)
	index.allHeaders = make(map[string]*ReferenceNode)
	index.allExamples = make(map[string]*ReferenceNode)
	index.allLinks = make(map[string]*ReferenceNode)
	index.allCallbacks = make(map[string]*ReferenceNode)
	index.allExternalDocuments = make(map[string]*ReferenceNode)
	index.securityRequirementRefs = make(map[string]map[string][]*ReferenceNode)
	index.polymorphicRefs = make(map[string]*ReferenceNode)
	index.refsWithSiblings = make(map[string]ReferenceNode)
	index.opServersRefs = make(map[string]map[string][]*ReferenceNode)
	index.componentIndexChan = make(chan bool)
	index.polyComponentIndexChan = make(chan bool)
}

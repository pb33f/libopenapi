// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"gopkg.in/yaml.v3"
	"net/http"
	"strings"
	"time"
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

func DetermineReferenceResolveType(ref string) int {
	if ref != "" && ref[0] == '#' {
		return LocalResolve
	}
	if ref != "" && len(ref) >= 5 && (ref[:5] == "https" || ref[:5] == "http:") {
		return HttpResolve
	}
	if strings.Contains(ref, ".json") ||
		strings.Contains(ref, ".yaml") ||
		strings.Contains(ref, ".yml") {
		return FileResolve
	}
	return -1
}

func boostrapIndexCollections(rootNode *yaml.Node, index *SpecIndex) {
	index.root = rootNode
	index.allRefs = make(map[string]*Reference)
	index.allMappedRefs = make(map[string]*Reference)
	index.refsByLine = make(map[string]map[int]bool)
	index.linesWithRefs = make(map[int]bool)
	index.pathRefs = make(map[string]map[string]*Reference)
	index.paramOpRefs = make(map[string]map[string]map[string][]*Reference)
	index.operationTagsRefs = make(map[string]map[string][]*Reference)
	index.operationDescriptionRefs = make(map[string]map[string]*Reference)
	index.operationSummaryRefs = make(map[string]map[string]*Reference)
	index.paramCompRefs = make(map[string]*Reference)
	index.paramAllRefs = make(map[string]*Reference)
	index.paramInlineDuplicateNames = make(map[string][]*Reference)
	index.globalTagRefs = make(map[string]*Reference)
	index.securitySchemeRefs = make(map[string]*Reference)
	index.requestBodiesRefs = make(map[string]*Reference)
	index.responsesRefs = make(map[string]*Reference)
	index.headersRefs = make(map[string]*Reference)
	index.examplesRefs = make(map[string]*Reference)
	index.callbacksRefs = make(map[string]map[string][]*Reference)
	index.linksRefs = make(map[string]map[string][]*Reference)
	index.callbackRefs = make(map[string]*Reference)
	index.externalSpecIndex = make(map[string]*SpecIndex)
	index.allComponentSchemaDefinitions = make(map[string]*Reference)
	index.allParameters = make(map[string]*Reference)
	index.allSecuritySchemes = make(map[string]*Reference)
	index.allRequestBodies = make(map[string]*Reference)
	index.allResponses = make(map[string]*Reference)
	index.allHeaders = make(map[string]*Reference)
	index.allExamples = make(map[string]*Reference)
	index.allLinks = make(map[string]*Reference)
	index.allCallbacks = make(map[string]*Reference)
	index.allExternalDocuments = make(map[string]*Reference)
	index.securityRequirementRefs = make(map[string]map[string][]*Reference)
	index.polymorphicRefs = make(map[string]*Reference)
	index.refsWithSiblings = make(map[string]Reference)
	index.seenRemoteSources = make(map[string]*yaml.Node)
	index.seenLocalSources = make(map[string]*yaml.Node)
	index.opServersRefs = make(map[string]map[string][]*Reference)
	index.httpClient = &http.Client{Timeout: time.Duration(5) * time.Second}
	index.componentIndexChan = make(chan bool)
	index.polyComponentIndexChan = make(chan bool)
}

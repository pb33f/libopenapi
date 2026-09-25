// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
)

// ClearAllCaches resets every global in-process cache in libopenapi.
//
// Calling it is not required to release memory: caches keyed by YAML nodes or model objects hold them weakly,
// so a document is reclaimed as soon as the caller drops it. Use it to force hashes to be recalculated after
// YAML nodes or low-level models were modified in place, or to empty the string-keyed caches (compiled JSONPath
// expressions, schema quick hashes and remote content types). It is safe to call while other goroutines parse,
// build or compare documents.
func ClearAllCaches() {
	low.ClearHashCache()              // model and YAML node hashes
	lowbase.ClearSchemaQuickHashMap() // SchemaQuickHashMap
	index.ClearHashCache()            // nodeHashCache
	index.ClearContentDetectionCache()
	highbase.ClearInlineRenderingTracker()
	high.ClearEncodeCache()
	utils.ClearJSONPathCache()
}

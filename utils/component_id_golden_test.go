// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils_test

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

// goldenCorpusPath is the byte-for-byte contract for ConvertComponentIdIntoFriendlyPathSearch.
// The file is gzipped; each line is: input<TAB>name<TAB>path. It is generated from real
// reference and schema definitions collected by indexing every spec in test_specs/, plus
// hand-picked edge cases, all run through the converter. Regenerate with:
//
//	GOLDEN_REGENERATE=true go test ./utils -run TestComponentIdGoldenCorpus
const goldenCorpusPath = "testdata/component_id_golden.txt.gz"

// goldenCorpusPerSpecCap bounds the number of unique inputs taken per spec so the
// fixture stays reviewable. Inputs are sorted before capping, so the selection is
// deterministic; the cap is recorded per spec during generation, never silent.
const goldenCorpusPerSpecCap = 4000

// goldenEdgeCaseInputs are converter inputs that exercise documented quirks regardless
// of whether any spec in test_specs happens to contain them.
var goldenEdgeCaseInputs = []string{
	"",
	"#/",
	"#/components/schemas/Burger",
	"#/components/schemas/Burger/properties/fries",
	"#/paths/~1burgers~1{burgerId}/get",
	"#/paths/~1burgers/post/responses/200",
	"#/components/schemas/List/items/0",
	"#/components/schemas/List/items/100",
	"#/components/schemas/403_permission_denied",
	"#/components/schemas/some%20space",
	"#/components/schemas/async_search.submit#wait_for_completion_timeout",
	"#/definitions/Pet",
	"document.yaml#/components/schemas/Thing",
	"/absolute/path/file.yaml#/components/schemas/Thing",
	"not-a-pointer",
	"#/a/b/c/d/e/f/g/h",
}

func collectGoldenCorpusInputs(t *testing.T) []string {
	specDir := "../test_specs"
	entries, err := os.ReadDir(specDir)
	require.NoError(t, err)

	seen := make(map[string]struct{})
	for _, in := range goldenEdgeCaseInputs {
		seen[in] = struct{}{}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".json" {
			continue
		}
		data, rErr := os.ReadFile(filepath.Join(specDir, entry.Name()))
		require.NoError(t, rErr)

		var rootNode yaml.Node
		if yaml.Unmarshal(data, &rootNode) != nil {
			continue // some fixtures are intentionally unparsable
		}
		if len(rootNode.Content) == 0 {
			continue
		}

		cfg := index.CreateOpenAPIIndexConfig()
		cfg.AllowRemoteLookup = false
		cfg.AllowFileLookup = false
		idx := index.NewSpecIndexWithConfig(&rootNode, cfg)

		specInputs := make(map[string]struct{})
		for _, ref := range idx.GetRawReferencesSequenced() {
			specInputs[ref.Definition] = struct{}{}
		}
		for _, ref := range idx.GetAllSequencedReferences() {
			specInputs[ref.Definition] = struct{}{}
		}
		for _, ref := range idx.GetAllInlineSchemas() {
			specInputs[ref.Definition] = struct{}{}
		}
		for _, ref := range idx.GetAllReferenceSchemas() {
			specInputs[ref.Definition] = struct{}{}
		}
		for def := range idx.GetMappedReferences() {
			specInputs[def] = struct{}{}
		}

		ordered := make([]string, 0, len(specInputs))
		for in := range specInputs {
			if strings.ContainsAny(in, "\t\n") {
				continue // the fixture format is tab separated lines
			}
			ordered = append(ordered, in)
		}
		sort.Strings(ordered)
		if len(ordered) > goldenCorpusPerSpecCap {
			// stride-sample across the sorted range so the selection spans the
			// whole spec rather than the alphabetically-first slice.
			t.Logf("golden corpus: sampling %s from %d down to %d inputs", entry.Name(), len(ordered), goldenCorpusPerSpecCap)
			sampled := make([]string, 0, goldenCorpusPerSpecCap)
			stride := float64(len(ordered)) / float64(goldenCorpusPerSpecCap)
			for i := 0; i < goldenCorpusPerSpecCap; i++ {
				sampled = append(sampled, ordered[int(float64(i)*stride)])
			}
			ordered = sampled
		}
		for _, in := range ordered {
			seen[in] = struct{}{}
		}
	}

	all := make([]string, 0, len(seen))
	for in := range seen {
		all = append(all, in)
	}
	sort.Strings(all)
	return all
}

func TestComponentIdGoldenCorpus(t *testing.T) {
	if os.Getenv("GOLDEN_REGENERATE") == "true" {
		inputs := collectGoldenCorpusInputs(t)
		require.NoError(t, os.MkdirAll("testdata", 0o755))
		out, cErr := os.Create(goldenCorpusPath)
		require.NoError(t, cErr)
		gz := gzip.NewWriter(out)
		for _, in := range inputs {
			name, path := utils.ConvertComponentIdIntoFriendlyPathSearch(in)
			_, _ = gz.Write([]byte(in))
			_, _ = gz.Write([]byte{'\t'})
			_, _ = gz.Write([]byte(name))
			_, _ = gz.Write([]byte{'\t'})
			_, _ = gz.Write([]byte(path))
			_, _ = gz.Write([]byte{'\n'})
		}
		require.NoError(t, gz.Close())
		require.NoError(t, out.Close())
		t.Logf("golden corpus regenerated: %d inputs", len(inputs))
	}

	f, err := os.Open(goldenCorpusPath)
	require.NoError(t, err, "golden corpus fixture missing - regenerate with GOLDEN_REGENERATE=true")
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gzr.Close()

	scanner := bufio.NewScanner(gzr)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	lines := 0
	for scanner.Scan() {
		lines++
		parts := strings.SplitN(scanner.Text(), "\t", 3)
		require.Len(t, parts, 3, fmt.Sprintf("malformed golden corpus line %d", lines))
		name, path := utils.ConvertComponentIdIntoFriendlyPathSearch(parts[0])
		assert.Equal(t, parts[1], name, "name mismatch for input %q (line %d)", parts[0], lines)
		assert.Equal(t, parts[2], path, "path mismatch for input %q (line %d)", parts[0], lines)
	}
	require.NoError(t, scanner.Err())
	assert.Greater(t, lines, 1000, "golden corpus suspiciously small")
}

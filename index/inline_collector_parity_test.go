// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// inlineCollectorParityPath pins the exact (Definition, FullDefinition, Path) tuples
// produced by the inline schema collectors for real documents. The converter golden
// corpus (utils/testdata) protects the path converter; this fixture protects the
// string assembly inside the collectors themselves. Regenerate with:
//
//	GOLDEN_REGENERATE=true go test ./index -run TestInlineCollectorParity
const inlineCollectorParityPath = "testdata/inline_collector_parity.txt"

// parityAbsolutePath is a fixed absolute path so fixtures are machine independent.
const parityAbsolutePath = "/parity/test/root.yaml"

var parityScanSpecs = []string{
	"../test_specs/stripe.yaml",
	"../test_specs/burgershop.openapi.yaml",
	"../test_specs/mixedref-burgershop.openapi.yaml",
	"../test_specs/k8s.json",
}

func collectInlineParityRows(t *testing.T) []string {
	var rows []string
	for _, spec := range parityScanSpecs {
		data, err := os.ReadFile(spec)
		require.NoError(t, err)

		var rootNode yaml.Node
		require.NoError(t, yaml.Unmarshal(data, &rootNode))

		cfg := CreateOpenAPIIndexConfig()
		cfg.AllowRemoteLookup = false
		cfg.AllowFileLookup = false
		cfg.SpecAbsolutePath = parityAbsolutePath
		idx := NewSpecIndexWithConfig(&rootNode, cfg)

		collections := []struct {
			name string
			refs []*Reference
		}{
			{"inline", idx.GetAllInlineSchemas()},
			{"refs", idx.GetAllReferenceSchemas()},
			{"objects", idx.GetAllInlineSchemaObjects()},
		}
		for _, c := range collections {
			tuples := make([]string, 0, len(c.refs))
			for _, ref := range c.refs {
				tuples = append(tuples, ref.Definition+"\x1f"+ref.FullDefinition+"\x1f"+ref.Path)
			}
			sort.Strings(tuples)
			h := sha256.Sum256([]byte(strings.Join(tuples, "\n")))
			rows = append(rows, fmt.Sprintf("%s\t%s\t%d\t%s",
				filepath.Base(spec), c.name, len(tuples), hex.EncodeToString(h[:])))
		}
	}
	return rows
}

func TestInlineCollectorParity(t *testing.T) {
	rows := collectInlineParityRows(t)

	if os.Getenv("GOLDEN_REGENERATE") == "true" {
		require.NoError(t, os.MkdirAll("testdata", 0o755))
		require.NoError(t, os.WriteFile(inlineCollectorParityPath,
			[]byte(strings.Join(rows, "\n")+"\n"), 0o644))
		t.Logf("inline collector parity fixture regenerated: %d rows", len(rows))
	}

	expected, err := os.ReadFile(inlineCollectorParityPath)
	require.NoError(t, err, "parity fixture missing - regenerate with GOLDEN_REGENERATE=true")
	assert.Equal(t, strings.TrimRight(string(expected), "\n"), strings.Join(rows, "\n"),
		"inline collector output changed - Definition/FullDefinition/Path must stay byte-identical")
}

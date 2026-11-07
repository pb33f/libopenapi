// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

func TestBundleBytesComposed_SiblingAlias(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		for _, origins := range []bool{false, true} {
			t.Run(fmt.Sprintf("explicit=%t/origins=%t", explicit, origins), func(t *testing.T) {
				root, err := filepath.Abs("test/specs/sibling_alias/root.json")
				require.NoError(t, err)
				data, err := os.ReadFile(root)
				require.NoError(t, err)
				var logs bytes.Buffer
				cfg := &datamodel.DocumentConfiguration{
					BasePath: filepath.Dir(root), ExtractRefsSequentially: true,
					AllowRemoteReferences: true, ExcludeExtensionRefs: true,
					Logger: slog.New(slog.NewTextHandler(&logs, nil)),
				}
				if explicit {
					cfg.SpecFilePath = root
					cfg.AllowFileReferences = true
				}
				var out []byte
				if origins {
					result, bundleErr := BundleBytesComposedWithOrigins(data, cfg, &BundleCompositionConfig{Delimiter: "__"})
					require.NoError(t, bundleErr)
					out = result.Bytes
				} else {
					out, err = BundleBytesComposed(data, cfg, &BundleCompositionConfig{Delimiter: "__"})
					require.NoError(t, err)
				}
				require.NotContains(t, logs.String(), "level=ERROR")
				doc, err := libopenapi.NewDocument(out)
				require.NoError(t, err)
				model, err := doc.BuildV3Model()
				require.NoError(t, err)
				for _, name := range []string{"output", "item", "price", "amount", "number"} {
					require.NotNil(t, model.Model.Components.Schemas.GetOrZero(name), name)
				}
				price := model.Model.Components.Schemas.GetOrZero("price").Schema()
				require.NotNil(t, price)
				amount := price.Properties.GetOrZero("amount").Schema()
				require.NotNil(t, amount)
				require.Equal(t, "Local description", amount.Description)
				require.Equal(t, "#/components/schemas/amount", price.Properties.GetOrZero("amount").GetReference())
				alias := model.Model.Components.Schemas.GetOrZero("amount").Schema()
				require.NotNil(t, alias)
				require.Equal(t, "Alias description", alias.Description)
				require.Equal(t, "#/components/schemas/number", model.Model.Components.Schemas.GetOrZero("amount").GetReference())
				require.Equal(t, []string{"number"}, model.Model.Components.Schemas.GetOrZero("number").Schema().Type)
				assertNoFilePathRefs(t, out)
			})
		}
	}
}

func TestBundleBytesComposed_SiblingAliasMissingTarget(t *testing.T) {
	tmp := t.TempDir()
	for _, file := range []string{"root.json", "parts.json", "common.json"} {
		data, err := os.ReadFile(filepath.Join("test/specs/sibling_alias", file))
		require.NoError(t, err)
		if file == "common.json" {
			data = bytes.ReplaceAll(data, []byte(`"#/number"`), []byte(`"#/missing"`))
		}
		require.NoError(t, os.WriteFile(filepath.Join(tmp, file), data, 0600))
	}
	data, err := os.ReadFile(filepath.Join(tmp, "root.json"))
	require.NoError(t, err)
	for _, origins := range []bool{false, true} {
		t.Run(fmt.Sprintf("origins=%t", origins), func(t *testing.T) {
			cfg := &datamodel.DocumentConfiguration{BasePath: tmp, SpecFilePath: filepath.Join(tmp, "root.json"), AllowFileReferences: true}
			if origins {
				_, err = BundleBytesComposedWithOrigins(data, cfg, nil)
			} else {
				_, err = BundleBytesComposed(data, cfg, nil)
			}
			require.ErrorContains(t, err, "missing")
		})
	}
}

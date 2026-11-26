// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

func loadMembers(t testing.TB, dir string, files map[string][]byte) *Document {
	t.Helper()
	d, err := LoadCollection(files, dir, nil)
	require.NoError(t, err)
	t.Cleanup(d.Release)
	return d
}

func sharedMembers(uses, minimum int) map[string][]byte {
	files := map[string][]byte{"money.json": []byte(fmt.Sprintf(`{"$id":"urn:money","properties":{"currency":{"type":"string","minLength":%d}}}`, minimum))}
	for i := range uses {
		files[fmt.Sprintf("orders/%04d.json", i)] = []byte(fmt.Sprintf(`{"$id":"urn:order:%d","properties":{"value":{"$ref":"urn:money"}}}`, i))
	}
	return files
}

func TestCollectionExternalBaseAfterMemberLookup(t *testing.T) {
	members := map[string][]byte{
		"order.json": []byte(`{"properties":{"price":{"$ref":"money.yaml"},"currency":{"$ref":"types/currency.yaml#code"}}}`),
		"money.yaml": []byte("type: integer"),
	}
	const currency = "$anchor: code\ntype: string"
	for _, remote := range []bool{false, true} {
		config := datamodel.NewDocumentConfiguration()
		config.BasePath = t.TempDir()
		config.LocalFS = fstest.MapFS{"types/currency.yaml": {Data: []byte(currency)}}
		if remote {
			config.BaseURL, _ = url.Parse("https://schemas.example.test/external/")
			config.AllowRemoteReferences = true
			config.RemoteURLHandler = func(location string) (*http.Response, error) {
				require.Equal(t, "https://schemas.example.test/external/types/currency.yaml", location)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(currency))}, nil
			}
		}
		doc, err := LoadCollection(members, t.TempDir(), config)
		require.NoError(t, err)
		require.Len(t, doc.References, 2)
		require.Same(t, doc.Sources["money.yaml"].Node, doc.References[0].To)
		require.Equal(t, "string", member(doc.References[1].To, "type").Value)
		doc.Release()
	}
}

func TestCollectionSharedOwnership(t *testing.T) {
	left, right := loadMembers(t, "original", sharedMembers(2, 2)), loadMembers(t, "modified", sharedMembers(2, 3))
	require.Nil(t, right.Root)
	require.Len(t, right.Sources, 3)
	require.Len(t, right.References, 2)
	require.Same(t, right.Sources["money.json"].Node, right.References[0].To)
	require.Same(t, right.References[0].To, right.References[1].To)
	diff, err := Compare(left, right, nil)
	require.NoError(t, err)
	require.Len(t, diff.GetAllChanges(), 1)
	ch := diff.GetAllChanges()[0]
	require.Equal(t, "urn:money", ch.SchemaResource)
	require.Equal(t, "$.properties.currency.minLength", ch.Path)
	require.True(t, ch.Breaking)
	require.Equal(t, "money.json", ch.Context.OriginalDocument)
	wire, err := json.Marshal(ch)
	require.NoError(t, err)
	require.Contains(t, string(wire), `"schemaResource":"urn:money"`)
	reverse, err := Compare(right, left, nil)
	require.NoError(t, err)
	require.Zero(t, reverse.TotalBreakingChanges())
}

func TestCollectionSourceFilesAndAuthoredCoordinates(t *testing.T) {
	old := loadMembers(t, "before", map[string][]byte{"a/money.yaml": []byte("$id: urn:money\ntype: string\n")})
	newer := loadMembers(t, "after", map[string][]byte{"b/money.yaml": []byte("$id: urn:money\ntype: string\nmaxLength: 12\n")})
	empty := loadMembers(t, "empty", nil)
	for _, tt := range []struct {
		name                  string
		left, right           *Document
		original, modified    string
		authored              string
		originalLine, newLine *int
	}{
		{"added constraint in moved file", old, newer, "a/money.yaml", "b/money.yaml", "b/money.yaml", nil, new(3)},
		{"removed constraint in moved file", newer, old, "b/money.yaml", "a/money.yaml", "b/money.yaml", new(3), nil},
		{"added file", empty, newer, "", "b/money.yaml", "b/money.yaml", nil, new(1)},
		{"removed file", newer, empty, "b/money.yaml", "", "b/money.yaml", new(1), nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := Compare(tt.left, tt.right, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)
			require.Len(t, diff.GetAllChanges(), 1)
			ctx := diff.GetAllChanges()[0].Context
			require.Equal(t, tt.original, ctx.OriginalDocument)
			require.Equal(t, tt.modified, ctx.ModifiedDocument)
			require.Equal(t, tt.authored, ctx.DocumentLocation)
			require.Equal(t, tt.originalLine, ctx.OriginalLine)
			require.Equal(t, tt.newLine, ctx.NewLine)
			require.Equal(t, tt.originalLine == nil, ctx.OriginalColumn == nil)
			require.Equal(t, tt.newLine == nil, ctx.NewColumn == nil)
		})
	}
}

func TestCollectionIdentityAndMoves(t *testing.T) {
	for _, tc := range []struct {
		name        string
		left, right map[string][]byte
		changes     int
	}{
		{"moved", map[string][]byte{"old.json": []byte(`{"$id":"urn:money","type":"string"}`)}, map[string][]byte{"dir/new.json": []byte(`{"$id":"urn:money","type":"string"}`)}, 0},
		{"anonymous", map[string][]byte{"a.yaml": []byte("minLength: 2")}, map[string][]byte{"a.yaml": []byte("minLength: 3")}, 1},
		{"rename id", map[string][]byte{"a.yaml": []byte("$id: urn:first")}, map[string][]byte{"a.yaml": []byte("$id: urn:second")}, 1},
		{"added", nil, map[string][]byte{"a.json": []byte("true")}, 1},
		{"removed", map[string][]byte{"a.json": []byte("false")}, nil, 1},
		{"empty", nil, nil, 0},
		{"distinct", map[string][]byte{"a.yaml": []byte("minLength: 1"), "b.yaml": []byte("minLength: 1")}, map[string][]byte{"a.yaml": []byte("minLength: 2"), "b.yaml": []byte("minLength: 2")}, 2},
		{"nested base", map[string][]byte{"a.json": []byte(`{"$id":"dir/a","$defs":{"field":{"$id":"field","minLength":1}},"$ref":"field"}`)}, map[string][]byte{"a.json": []byte(`{"$id":"dir/a","$defs":{"field":{"$id":"field","minLength":2}},"$ref":"field"}`)}, 1},
		{"retarget", map[string][]byte{"a.yaml": []byte("$ref: urn:one"), "one.yaml": []byte("$id: urn:one"), "two.yaml": []byte("$id: urn:two")}, map[string][]byte{"a.yaml": []byte("$ref: urn:two"), "one.yaml": []byte("$id: urn:one"), "two.yaml": []byte("$id: urn:two")}, 1},
		{"cycle", map[string][]byte{"a.json": []byte(`{"$ref":"b.json"}`), "b.json": []byte(`{"$ref":"a.json","minLength":2}`)}, map[string][]byte{"a.json": []byte(`{"$ref":"b.json"}`), "b.json": []byte(`{"$ref":"a.json","minLength":3}`)}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := loadMembers(t, "previous", tc.left), loadMembers(t, "current", tc.right)
			diff, err := Compare(a, b, nil)
			require.NoError(t, err)
			if tc.changes == 0 {
				require.Nil(t, diff)
				return
			}
			require.NotNil(t, diff)
			require.Equal(t, tc.changes, diff.TotalChanges())
			if tc.name == "distinct" {
				cs := diff.GetAllChanges()
				require.Equal(t, []string{"a.yaml", "b.yaml"}, []string{cs[0].SchemaResource, cs[1].SchemaResource})
				require.Equal(t, cs[0].Path, cs[1].Path)
			}
			if tc.name == "nested base" {
				require.Equal(t, "dir/field", diff.GetAllChanges()[0].SchemaResource)
			}
		})
	}
}

func TestMatchResourcesPrefersIDThenOrigin(t *testing.T) {
	left := loadMembers(t, "previous", map[string][]byte{
		"a.yml": []byte("$id: urn:move"), "b.yml": []byte("$id: urn:rename"), "gone.yml": []byte("true"),
	})
	right := loadMembers(t, "current", map[string][]byte{
		"new.yml": []byte("$id: urn:move"), "b.yml": []byte("$id: urn:renamed"), "fresh.yml": []byte("false"),
	})
	pairs := MatchResources(left, right)
	require.Len(t, pairs, 2)
	for old, current := range pairs {
		want := map[string]string{"urn:move": "urn:move", "urn:rename": "urn:renamed"}
		require.Equal(t, want[old.ID], current.ID)
	}
}

func TestCollectionDuplicateIDAndRetrievalAlias(t *testing.T) {
	_, err := LoadCollection(map[string][]byte{"a.json": []byte(`{"$id":"urn:money"}`), "b.json": []byte(`{"$id":"urn:money"}`)}, "root", nil)
	require.ErrorContains(t, err, "duplicate")
	require.Contains(t, err.Error(), "a.json")
	require.Contains(t, err.Error(), "b.json")
	d := loadMembers(t, "root", map[string][]byte{"a.json": []byte(`{"$id":"urn:a?x=1"}`), "b.json": []byte(`{"$id":"urn:a?x=2"}`), "refs.json": []byte(`{"properties":{"v":{"$ref":"b.json"}}}`)})
	require.NotEqual(t, d.Resources[0].ID, d.Resources[1].ID)
	require.Same(t, d.Sources["b.json"].Node, d.References[0].To)
	for _, name := range []string{".", "../a.json", "dir\\a.json", "/outside.json"} {
		_, err = LoadCollection(map[string][]byte{name: []byte("true")}, "root", nil)
		require.ErrorContains(t, err, "invalid schema member")
	}
}

// BenchmarkSharedReferences measures the complete authored comparison after indexing.
// Every incoming use points at the same target node and source buffer on its side.
func BenchmarkSharedReferences(b *testing.B) {
	for _, uses := range []int{2, 100, 1000} {
		b.Run(fmt.Sprint(uses), func(b *testing.B) {
			left, right := loadMembers(b, "original", sharedMembers(uses, 2)), loadMembers(b, "modified", sharedMembers(uses, 3))
			for _, doc := range []*Document{left, right} {
				for _, ref := range doc.References {
					if ref.To != doc.Sources["money.json"].Node {
						b.Fatal("multiple target nodes")
					}
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				diff, err := Compare(left, right, nil)
				if err != nil || diff == nil || diff.TotalChanges() != 1 || !strings.HasPrefix(diff.GetAllChanges()[0].Path, "$.properties.currency.") {
					b.Fatalf("unexpected definition edit: %v", err)
				}
			}
			b.ReportMetric(1, "targetEdits/op")
			b.ReportMetric(float64(uses+1), "sources/side")
		})
	}
}

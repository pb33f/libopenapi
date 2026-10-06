package orderedmap

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high/nodes"
	"github.com/pb33f/testify/require"
)

// untypedKey mirrors low.KeyReference: a struct key exposing its string through GetValueUntyped.
type untypedKey struct {
	Value string
	Node  *yaml.Node
}

func (k untypedKey) GetValueUntyped() any { return k.Value }

// stringerKey is a struct whose String method takes over %v formatting.
type stringerKey struct{ Value string }

func (k stringerKey) String() string { return "s:" + k.Value }

// plainKey is a struct with no methods at all, printed as "{...}" by %v.
type plainKey struct{ Value string }

// errorKey and formatterKey take over %v formatting through error and fmt.Formatter.
type errorKey struct{ Value string }

func (k errorKey) Error() string { return k.Value }

type formatterKey struct{ Value string }

func (k formatterKey) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(k.Value)) }

// requireIndexMatchesScan asserts the index answers every probe exactly as the FindValueUntyped scan does.
func requireIndexMatchesScan[K comparable, V any](t *testing.T, m *Map[K, V], probes []string) {
	t.Helper()
	x := m.untypedValueIndex()
	for _, probe := range probes {
		require.Equal(t, m.FindValueUntyped(probe), x.find(probe), "probe %q", probe)
	}
}

func indexProbes(n int) []string {
	probes := []string{"", "missing", "{", "{key-0", "s:key-1"}
	for i := 0; i < n; i++ {
		probes = append(probes, "key-"+strconv.Itoa(i))
	}
	return probes
}

func TestUntypedValueIndex_MatchesScan(t *testing.T) {
	for _, n := range []int{0, 1, 5, indexScanLimit, indexScanLimit + 1, 64} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			probes := indexProbes(n)

			strings := New[string, int]()
			structs := New[untypedKey, int]()
			pointers := New[*untypedKey, int]()
			stringers := New[stringerKey, int]()
			plains := New[plainKey, int]()
			for i := 0; i < n; i++ {
				name := "key-" + strconv.Itoa(i)
				strings.Set(name, i)
				structs.Set(untypedKey{Value: name, Node: &yaml.Node{}}, i)
				pointers.Set(&untypedKey{Value: name}, i)
				stringers.Set(stringerKey{Value: name}, i)
				plains.Set(plainKey{Value: name}, i)
			}
			// the %v forms of these keys are only knowable at runtime, so probe for them too.
			for k := range structs.KeysFromOldest() {
				probes = append(probes, fmt.Sprintf("%v", k))
			}
			for k := range pointers.KeysFromOldest() {
				probes = append(probes, fmt.Sprintf("%v", k))
			}
			for k := range plains.KeysFromOldest() {
				probes = append(probes, fmt.Sprintf("%v", k))
			}

			requireIndexMatchesScan(t, strings, probes)
			requireIndexMatchesScan(t, structs, probes)
			requireIndexMatchesScan(t, pointers, probes)
			requireIndexMatchesScan(t, stringers, probes)
			requireIndexMatchesScan(t, plains, probes)
		})
	}
}

// When two pairs produce the same match string, the oldest pair wins, as it does for the scan.
func TestUntypedValueIndex_OldestMatchWins(t *testing.T) {
	for _, n := range []int{2, indexScanLimit + 4} {
		m := New[untypedKey, int]()
		for i := 0; i < n; i++ {
			m.Set(untypedKey{Value: "same", Node: &yaml.Node{Line: i}}, i)
		}
		require.Equal(t, 0, m.untypedValueIndex().find("same"))
		requireIndexMatchesScan(t, m, []string{"same", "other"})
	}
}

// A nil map finds nothing, matching FindValueUntyped on a nil map.
func TestUntypedValueIndex_NilMap(t *testing.T) {
	var m *Map[untypedKey, int]
	x := m.untypedValueIndex()
	require.Nil(t, x.find("anything"))
	require.Nil(t, m.FindValueUntyped("anything"))
}

func TestFormatsAsStructLiteral(t *testing.T) {
	require.True(t, formatsAsStructLiteral(reflect.TypeFor[untypedKey]()))
	require.True(t, formatsAsStructLiteral(reflect.TypeFor[plainKey]()))
	require.False(t, formatsAsStructLiteral(reflect.TypeFor[*untypedKey]()))
	require.False(t, formatsAsStructLiteral(reflect.TypeFor[string]()))
	require.False(t, formatsAsStructLiteral(reflect.TypeFor[stringerKey]()))
	require.False(t, formatsAsStructLiteral(reflect.TypeFor[errorKey]()))
	require.False(t, formatsAsStructLiteral(reflect.TypeFor[formatterKey]()))
}

func TestKeyNodeIndex_MatchesFindKeyNode(t *testing.T) {
	require.Nil(t, (&keyNodeIndex{}).find("anything"))

	for _, n := range []int{1, indexScanLimit, indexScanLimit + 1, 64} {
		mapNode := &yaml.Node{Kind: yaml.MappingNode}
		for i := 0; i < n; i++ {
			// every key appears twice, so first-match semantics are exercised.
			for _, dup := range []int{0, 1} {
				mapNode.Content = append(mapNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "key-" + strconv.Itoa(i), Line: dup},
					&yaml.Node{Kind: yaml.ScalarNode, Value: "value"})
			}
		}
		x := &keyNodeIndex{mapNode: mapNode}
		for _, probe := range append(indexProbes(n), "value") {
			require.Same(t, findKeyNode(probe, mapNode), x.find(probe), "probe %q", probe)
		}
	}
}

// scanOnlyLookup implements FindValueUntyped without the index hook, as a type outside this package would.
type scanOnlyLookup struct{ values map[string]any }

func (s scanOnlyLookup) FindValueUntyped(key string) any { return s.values[key] }

type untypedHolder struct {
	value any
	node  *yaml.Node
}

func (h untypedHolder) GetValueUntyped() any     { return h.value }
func (h untypedHolder) GetValueNode() *yaml.Node { return h.node }

type recordingBuilder struct{ lowValues map[string]any }

func (r *recordingBuilder) AddYAMLNode(parent *yaml.Node, entry *nodes.NodeEntry) *yaml.Node {
	r.lowValues[entry.Key] = entry.LowValue
	return parent
}

// ToYamlNode resolves low values through the index when the low map provides one, and through
// FindValueUntyped when it does not.
func TestToYamlNode_LowValueResolution(t *testing.T) {
	high := New[string, string]()
	for i := 0; i < indexScanLimit+2; i++ {
		high.Set("key-"+strconv.Itoa(i), "v")
	}

	lowMap := New[untypedKey, int]()
	for i := 0; i < indexScanLimit+2; i++ {
		lowMap.Set(untypedKey{Value: "key-" + strconv.Itoa(i)}, i)
	}

	indexed := &recordingBuilder{lowValues: map[string]any{}}
	high.ToYamlNode(indexed, untypedHolder{value: lowMap})
	scanned := &recordingBuilder{lowValues: map[string]any{}}
	high.ToYamlNode(scanned, untypedHolder{value: scanOnlyLookup{values: map[string]any{"key-3": 3}}})
	nilLow := &recordingBuilder{lowValues: map[string]any{}}
	high.ToYamlNode(nilLow, untypedHolder{value: (*Map[untypedKey, int])(nil)})

	for i := 0; i < indexScanLimit+2; i++ {
		key := "key-" + strconv.Itoa(i)
		require.Equal(t, i, indexed.lowValues[key])
		require.Nil(t, nilLow.lowValues[key])
	}
	require.Equal(t, 3, scanned.lowValues["key-3"])
	require.Nil(t, scanned.lowValues["key-4"])
}

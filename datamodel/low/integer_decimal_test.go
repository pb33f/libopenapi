package low

import (
	"math"
	"strconv"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestBuildModel_ExactDecimalIntegers(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  int64
		set   bool
	}{
		{"9007199254740993.0", 9007199254740993, true},
		{"9.007199254740993e15", 9007199254740993, true},
		{"-9.007199254740993e15", -9007199254740993, true},
		{"9007199254740992.5", 0, false},
		{"1.0000000000000001", 0, false},
		{"1e-999999999999999999999", 0, false},
		{"0e-999999999999999999999", 0, true},
		{"1e999999999999999999999", math.MaxInt64, true},
		{"-1e999999999999999999999", math.MinInt64, true},
		{"9223372036854775807.0", math.MaxInt64, true},
		{"-9223372036854775808.0", math.MinInt64, true},
		{"9223372036854775808.0", math.MaxInt64, true},
		{"-9223372036854775809.0", math.MinInt64, true},
		{"001.2300e2", 123, true},
		{"1_000.00", 1000, true},
		{"100.0e-2", 1, true},
		{"100.0e-3", 0, false},
		{".0", 0, true},
		{"Inf", math.MaxInt64, true},
		{"-Inf", math.MinInt64, true},
		{"0x1p4", 16, true},
		{"0x1p64", math.MaxInt64, true},
		{"-0x1p64", math.MinInt64, true},
		{"0x1.8p0", 0, false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			// Explicit tags also exercise overflow literals that YAML's implicit
			// float resolver classifies as strings.
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte("maxLength: !!float "+tc.value), &root))
			var model struct{ MaxLength NodeReference[int64] }
			require.NoError(t, BuildModel(root.Content[0], &model))
			require.Equal(t, tc.set, !model.MaxLength.IsEmpty())
			require.Equal(t, tc.want, model.MaxLength.Value)
			got, set := parseIntNode(tc.value)
			require.Equal(t, tc.set, set)
			if tc.set && strconv.IntSize == 64 {
				require.Equal(t, tc.want, int64(got))
			}
		})
	}
}

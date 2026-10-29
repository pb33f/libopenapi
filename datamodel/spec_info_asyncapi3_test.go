package datamodel

import (
	"github.com/pb33f/testify/require"
	"testing"
)

func TestExtractSpecInfoAsyncAPI3(t *testing.T) {
	for _, spec := range []string{"asyncapi: 3.0.0\ninfo: {title: Events, version: 1.0.0}\n", `{"asyncapi":"3.0.0","info":{"title":"Events","version":"1.0.0"}}`} {
		info, err := ExtractSpecInfo([]byte(spec))
		require.NoError(t, err)
		require.Equal(t, "asyncapi", info.SpecType)
		require.Equal(t, "3.0.0", info.Version)
		require.NotNil(t, info.RootNode)
	}
	_, err := ExtractSpecInfo([]byte("asyncapi: 3.0.0\nasyncapi: 3.0.0\n"))
	require.Error(t, err)
}

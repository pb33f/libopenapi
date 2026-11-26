// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"net/url"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestFileURIVolumeAndPathSyntax(t *testing.T) {
	for _, tc := range []struct {
		path, volume, uri, native string
		separator                 byte
	}{
		{"/catalog/a b.json", "", "file:///catalog/a%20b.json", "/catalog/a b.json", '/'},
		{"//catalog/share/a.json", "", "file:////catalog/share/a.json", "//catalog/share/a.json", '/'},
		{"/C:/catalog/a.json", "", "file:///C:/catalog/a.json", "/C:/catalog/a.json", '/'},
		{"C:/catalog/a.json", "C:", "file:///C:/catalog/a.json", `C:\catalog\a.json`, '\\'},
		{"//server/share/a.json", "//server/share", "file://server/share/a.json", `\\server\share\a.json`, '\\'},
	} {
		u := volumeFileURL(tc.path, tc.volume)
		require.Equal(t, tc.uri, u.String())
		require.Equal(t, tc.native, nativeFilePath(u, tc.separator))
	}
	localhost, err := url.Parse("file://localhost/D:/catalog/a.json")
	require.NoError(t, err)
	require.Equal(t, `D:\catalog\a.json`, nativeFilePath(localhost, '\\'))
}

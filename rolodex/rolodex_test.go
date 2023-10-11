// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rolodex

import (
    "github.com/stretchr/testify/assert"
    "testing"
    "testing/fstest"
    "time"
)

func TestRolodex_LocalNativeFS(t *testing.T) {

    t.Parallel()
    testFS := fstest.MapFS{
        "spec.yaml":             {Data: []byte("hip"), ModTime: time.Now()},
        "subfolder/spec1.json":  {Data: []byte("hop"), ModTime: time.Now()},
        "subfolder2/spec2.yaml": {Data: []byte("chop"), ModTime: time.Now()},
        "subfolder2/hello.jpg":  {Data: []byte("shop"), ModTime: time.Now()},
    }

    baseDir := "/tmp"

    fileFS, err := NewLocalFS(baseDir, testFS)
    if err != nil {
        t.Fatal(err)
    }

    rolo := NewRolodex()
    rolo.AddLocalFS(baseDir, fileFS)

    f, rerr := rolo.Open("spec.yaml")
    assert.NoError(t, rerr)
    assert.Equal(t, "hip", f.GetContent())

}

func TestRolodex_LocalNonNativeFS(t *testing.T) {

    t.Parallel()
    testFS := fstest.MapFS{
        "spec.yaml":             {Data: []byte("hip"), ModTime: time.Now()},
        "subfolder/spec1.json":  {Data: []byte("hop"), ModTime: time.Now()},
        "subfolder2/spec2.yaml": {Data: []byte("chop"), ModTime: time.Now()},
        "subfolder2/hello.jpg":  {Data: []byte("shop"), ModTime: time.Now()},
    }

    baseDir := ""

    rolo := NewRolodex()
    rolo.AddLocalFS(baseDir, testFS)

    f, rerr := rolo.Open("spec.yaml")
    assert.NoError(t, rerr)

    assert.Equal(t, "hip", f.GetContent())
}

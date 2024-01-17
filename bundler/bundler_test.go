// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"bytes"
	"encoding/json"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleDocument_DigitalOcean(t *testing.T) {

	// test the mother of all exploded specs.
	tmp, _ := os.MkdirTemp("", "openapi")
	cmd := exec.Command("git", "clone", "https://github.com/digitalocean/openapi", tmp)
	defer os.RemoveAll(filepath.Join(tmp, "openapi"))

	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}

	spec, _ := filepath.Abs(filepath.Join(tmp+"/specification", "DigitalOcean-public.v2.yaml"))
	digi, _ := os.ReadFile(spec)

	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(digi), &datamodel.DocumentConfiguration{
		BasePath:                tmp + "/specification",
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})),
	})
	if err != nil {
		panic(err)
	}

	v3Doc, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		panic(errs)
	}

	bytes, e := BundleDocument(&v3Doc.Model)

	assert.NoError(t, e)
	assert.False(t, strings.Contains("$ref", string(bytes)), "should not contain $ref")

}

func TestBundleDocument_Circular(t *testing.T) {

	digi, _ := os.ReadFile("../test_specs/circular-tests.yaml")

	var logs []byte
	byteBuf := bytes.NewBuffer(logs)

	config := &datamodel.DocumentConfiguration{
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(byteBuf, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})),
	}
	doc, err := libopenapi.NewDocumentWithConfiguration(digi, config)
	if err != nil {
		panic(err)
	}

	v3Doc, errs := doc.BuildV3Model()

	// three circular ref issues.
	assert.Len(t, errs, 3)

	bytes, e := BundleDocument(&v3Doc.Model)
	assert.NoError(t, e)
	assert.Len(t, bytes, 3069)

	logEntries := strings.Split(byteBuf.String(), "\n")

	assert.Len(t, logEntries, 5)
	for _, entry := range logEntries {
		items := make(map[string]any)
		if entry != "" {
			_ = json.Unmarshal([]byte(entry), &items)
			assert.Equal(t, "[bundler] skipping circular reference", items["msg"])
		}
	}
	assert.NoError(t, e)
}

func TestBundleBytes(t *testing.T) {

	digi, _ := os.ReadFile("../test_specs/circular-tests.yaml")

	var logs []byte
	byteBuf := bytes.NewBuffer(logs)

	config := &datamodel.DocumentConfiguration{
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(byteBuf, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})),
	}

	bytes, e := BundleBytes(digi, config)
	assert.Error(t, e)
	assert.Len(t, bytes, 3069)

	logEntries := strings.Split(byteBuf.String(), "\n")

	assert.Len(t, logEntries, 5)
	for _, entry := range logEntries {
		items := make(map[string]any)
		if entry != "" {
			_ = json.Unmarshal([]byte(entry), &items)
			assert.Equal(t, "[bundler] skipping circular reference", items["msg"])
		}
	}
}

func TestBundleBytes_Bad(t *testing.T) {
	bytes, e := BundleBytes(nil, nil)
	assert.Error(t, e)
	assert.Nil(t, bytes)
}

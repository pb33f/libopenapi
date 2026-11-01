package index

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

const nestedRemoteFilesBase = "https://nested.test/"

func nestedRemoteFileRef(n int) string {
	return fmt.Sprintf("components:\n  schemas:\n    A:\n      $ref: '%sf%d.yaml#/components/schemas/A'\n", nestedRemoteFilesBase, n)
}

// indexNestedRemoteFiles indexes a root document followed by a chain of remote files f1 to
// fDepth, where each file references the next, so each one is indexed while the ones before
// it still are. It returns the number of files fetched and the result of indexing.
func indexNestedRemoteFiles(depth int, lastFile string) (int64, error) {
	var fetched atomic.Int64
	cfg := CreateOpenAPIIndexConfig()
	cfg.RemoteURLHandler = func(url string) (*http.Response, error) {
		name := strings.TrimSuffix(strings.TrimPrefix(url, nestedRemoteFilesBase+"f"), ".yaml")
		n, err := strconv.Atoi(name)
		if err != nil {
			return nil, fmt.Errorf("unexpected url %q", url)
		}
		fetched.Add(1)
		doc := lastFile
		if n < depth {
			doc = nestedRemoteFileRef(n + 1)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(doc)),
		}, nil
	}

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(nestedRemoteFileRef(1)), &root); err != nil {
		return 0, err
	}

	rolodex := NewRolodex(cfg)
	remoteFS, err := NewRemoteFSWithConfig(cfg)
	if err != nil {
		return 0, err
	}
	rolodex.AddRemoteFS(nestedRemoteFilesBase, remoteFS)
	rolodex.SetRootNode(&root)
	err = rolodex.IndexTheRolodex(context.Background())
	return fetched.Load(), err
}

const nestedRemoteLeaf = "components:\n  schemas:\n    A:\n      type: string\n"

func TestRolodex_NestedRemoteFiles_IndexesEveryFile(t *testing.T) {
	fetched, err := indexNestedRemoteFiles(50, nestedRemoteLeaf)
	require.NoError(t, err)
	assert.Equal(t, int64(50), fetched)
}

func TestRolodex_CircularRemoteFiles_DoNotDeadlock(t *testing.T) {
	done := make(chan int64, 1)
	go func() {
		// The result of indexing depends on circular reference handling; only finishing matters here.
		fetched, _ := indexNestedRemoteFiles(3, nestedRemoteFileRef(1))
		done <- fetched
	}()

	select {
	case fetched := <-done:
		assert.Equal(t, int64(3), fetched)
	case <-time.After(30 * time.Second):
		t.Fatal("indexing circular remote files did not finish")
	}
}

func BenchmarkRolodex_NestedRemoteFiles(b *testing.B) {
	for _, depth := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				fetched, err := indexNestedRemoteFiles(depth, nestedRemoteLeaf)
				if err != nil {
					b.Fatal(err)
				}
				if fetched != int64(depth) {
					b.Fatalf("fetched %d files, want %d", fetched, depth)
				}
			}
		})
	}
}

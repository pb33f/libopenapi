# Migrating

This file lists breaking changes to the libopenapi Go API and how to update code for each one.

## Upgrading past v0.40

This release makes two changes to the low-level model API. They don't change parsing, rendering, bundling or
what-changed output.

You don't need to change anything if your code only uses:
- the high-level model
- `GetNodes()`
- `AddNode()`

### `NodeMap.Nodes` is now a `*low.NodeLines`

Every low-level model embeds `low.NodeMap`, which records the YAML nodes the model was built from, keyed by line
number. `Nodes` was a `*sync.Map` keyed by `int`. It is now a `*low.NodeLines`. A `NodeLines` records writes as they
happen and builds its line index the first time it is read. Most models are built and never read, so building a
document and comparing documents with what-changed allocate less.

| v0.40 | now |
|---|---|
| `Nodes *sync.Map` | `Nodes *low.NodeLines` |
| `Nodes.Range(func(key, value any) bool)` | `Nodes.Range(func(line int, value any) bool)`, visiting lines in ascending order |
| `Nodes.Load(key any) (any, bool)` | `Nodes.Load(line int) (any, bool)` |
| `Nodes.Store(key, value any)` | `Nodes.Store(line int, value any)` |
| `low.NodeMap{Nodes: &sync.Map{}}` | `low.NodeMap{Nodes: &low.NodeLines{}}` (the zero value is ready to use) |
| `low.ExtractNodes(ctx, root) *sync.Map` | returns `*low.NodeLines` |
| `low.ExtractNodesRecursive(ctx, root) *sync.Map` | returns `*low.NodeLines` |
| `low.ExtractExtensionNodes(ctx, extensions, nodes *sync.Map)` | takes `*low.NodeLines` |
| `low.MergeRecursiveNodesIfLineAbsent(dst *sync.Map, node)` | takes `*low.NodeLines` |

`NodeLines` does not have `sync.Map`'s other methods:
- `Delete` and `Clear`
- `LoadOrStore` and `LoadAndDelete`
- `Swap`, `CompareAndSwap` and `CompareAndDelete`

A line's value is a `*yaml.Node`, or a `[]*yaml.Node` when several nodes share the line, as before.

This program reads the nodes of an `info` object:

```go
package main

import (
	"fmt"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
)

const spec = `openapi: 3.1.0
info:
  title: Burger Shop
  version: 1.0.0
paths: {}
`

func main() {
	doc, err := libopenapi.NewDocument([]byte(spec))
	if err != nil {
		panic(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		panic(err)
	}
	info := model.Model.GoLow().Info.Value

	// Range passes each line as an int, in ascending order. A line holds a *yaml.Node, or a
	// []*yaml.Node when several nodes share it.
	info.Nodes.Range(func(line int, value any) bool {
		switch v := value.(type) {
		case *yaml.Node:
			fmt.Printf("line %d: %s\n", line, v.Value)
		case []*yaml.Node:
			for _, n := range v {
				fmt.Printf("line %d: %s\n", line, n.Value)
			}
		}
		return true
	})

	// Load takes the line as an int.
	if value, ok := info.Nodes.Load(4); ok {
		fmt.Printf("line 4 holds %d nodes\n", len(value.([]*yaml.Node)))
	}

	// GetNodes is unchanged.
	fmt.Printf("GetNodes: %d lines\n", len(info.GetNodes()))
}
```

Output:

```
line 3: title
line 3: Burger Shop
line 4: version
line 4: 1.0.0
line 4 holds 2 nodes
GetNodes: 2 lines
```

### `NodeReference.Context` is removed

`low.NodeReference[T]` had a `Context` field. libopenapi only ever set it on the operations of a `PathItem`:
- `Get`, `Put`, `Post`, `Delete`, `Options`, `Head`, `Patch`, `Trace` and `Query`
- the operations in `AdditionalOperations`

On those references it held the context the operation was resolved and built with. On every other reference it
was nil.

Read that context from the operation instead:

| v0.40 | now |
|---|---|
| `pathItem.Get.Context` | `pathItem.Get.Value.GetContext()` |

`GetContext()` returns the context passed to the operation's `Build`, which is the context `Context` held. If an
operation is a `$ref` to another file, that context carries the other file's index.

If your code sets `Context` on references it creates, keep the context next to the reference in a type of your
own, for example `struct { Ref low.NodeReference[T]; Ctx context.Context }`.

This program reads the context the `get` operation of `/burgers` was built with:

```go
package main

import (
	"fmt"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/index"
)

const spec = `openapi: 3.1.0
info:
  title: Burger Shop
  version: 1.0.0
paths:
  /burgers:
    get:
      operationId: listBurgers
      responses:
        '200':
          description: OK
`

func main() {
	doc, err := libopenapi.NewDocument([]byte(spec))
	if err != nil {
		panic(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		panic(err)
	}
	pathItem := model.Model.GoLow().Paths.Value.FindPath("/burgers").Value

	// was: ctx := pathItem.Get.Context
	ctx := pathItem.Get.Value.GetContext()

	idx := ctx.Value(index.FoundIndexKey).(*index.SpecIndex)
	fmt.Println("operation:", pathItem.Get.Value.OperationId.Value)
	fmt.Println("built with the operation's index:", idx == pathItem.Get.Value.GetIndex())
}
```

Output:

```
operation: listBurgers
built with the operation's index: true
```

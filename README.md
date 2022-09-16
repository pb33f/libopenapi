# libopenapi - enterprise grade OpenAPI tools for golang.

![Pipeline](https://github.com/pb33f/libopenapi/workflows/Build/badge.svg)
![GoReportCard](https://goreportcard.com/badge/github.com/pb33f/libopenapi)
[![codecov](https://codecov.io/gh/pb33f/libopenapi/branch/main/graph/badge.svg?)](https://codecov.io/gh/pb33f/libopenapi)

libopenapi has full support for Swagger (OpenAPI 2), OpenAPI 3, and OpenAPI 3.1.

## Introduction - Why?

There is already a great OpenAPI library for golang, it's called [kin-openapi](https://github.com/getkin/kin-openapi).

### So why does this exist?

[kin-openapi](https://github.com/getkin/kin-openapi) is great, and you should use it. 

**However, it's missing one critical feature**

When building tooling that needs to analyze OpenAPI specifications at a *low* level, [kin-openapi](https://github.com/getkin/kin-openapi)
**runs out of power** when you need to know the original line numbers and columns, or comments within all keys and values in the spec.

All that data is **lost** when the spec is loaded in by [kin-openapi](https://github.com/getkin/kin-openapi). It's mainly
because the library will unmarshal data directly into structs, which works great - if you don't need access to the original 
specification low level details.

## libopenapi retains _everything_.

libopenapi has been designed to retain all of that really low-level detail about the AST, line numbers, column numbers, comments, 
original AST structure - everything you could need.

libopenapi has a **porcelain** (high-level) and a **plumbing** (low-level) API. Every high level struct, has the 
ability to `GoLow` and dive from the high-level model, down to the low-level model and look-up any detail about the 
underlying raw data backing that model.

This library exists because this very need existed inside [VMware](https://vmware.com), we built our own internal 
version of libopenapi, which isn't something that can be released as it's bespoke.

libopenapi is the result of years of learning and battle testing OpenAPI in golang. This library represents what we
would have created, if we knew then - what we know now.

> If you need to know which line, or column a key or value for something is? **libopenapi has you covered**

## Comes with an Index and a Resolver

Want a lightning fast way to look up any element in an OpenAPI swagger spec? **libopenapi has you covered**

Need a way to 'resolve' OpenAPI documents that are exploded out across multiple files, remotely or locally? 
**libopenapi has you covered**

---

## Some examples to get you started.

Grab the latest release of **libopenapi**

```
go get github.com/pb33f/libopenapi
```

### Load an OpenAPI 3+ spec into a model

```go
// load an OpenAPI 3 specification from bytes
petstore, _ := ioutil.ReadFile("test_specs/petstorev3.json")

// create a new document from specification bytes
document, err := NewDocument(petstore)

// if anything went wrong, an error is thrown
if err != nil {
    panic(fmt.Sprintf("cannot create new document: %e", err))
}

// because we know this is a v3 spec, we can build a ready to go model from it.
v3Model, errors := document.BuildV3Model()

// if anything went wrong when building the v3 model, a slice of errors will be returned
if len(errors) > 0 {
    for i := range errors {
        fmt.Printf("error: %e\n", errors[i])
    }
    panic(fmt.Sprintf("cannot create v3 model from document: %d errors reported", len(errors)))
}

// get a count of the number of paths and schemas.
paths := len(v3Model.Model.Paths.PathItems)
schemas := len(v3Model.Model.Components.Schemas)

// print the number of paths and schemas in the document
fmt.Printf("There are %d paths and %d schemas in the document", paths, schemas)
```

This will print: 

```
There are 13 paths and 8 schemas in the document
```


### Load a Swagger (OpenAPI 2) spec into a model
```go
// load a Swagger specification from bytes
petstore, _ := ioutil.ReadFile("test_specs/petstorev2.json")

// create a new document from specification bytes
document, err := NewDocument(petstore)

// if anything went wrong, an error is thrown
if err != nil {
    panic(fmt.Sprintf("cannot create new document: %e", err))
}

// because we know this is a v2 spec, we can build a ready to go model from it.
v2Model, errors := document.BuildV2Model()

// if anything went wrong when building the v3 model, a slice of errors will be returned
if len(errors) > 0 {
    for i := range errors {
        fmt.Printf("error: %e\n", errors[i])
    }
    panic(fmt.Sprintf("cannot create v3 model from document: %d errors reported", len(errors)))
}

// get a count of the number of paths and schemas.
paths := len(v2Model.Model.Paths.PathItems)
schemas := len(v2Model.Model.Definitions.Definitions)

// print the number of paths and schemas in the document
fmt.Printf("There are %d paths and %d schemas in the document", paths, schemas)
```

This will print: 

```
There are 14 paths and 6 schemas in the document
```

### Dropping down from the high-level API to the low-level one

This example shows how after loading an OpenAPI spec into a document, navigating to an Operation is pretty simple. 
It then shows how to _drop-down_ (using `GoLow())` to the low-level API and query the line and start column of the RequestBody description.

```go
// load an OpenAPI 3 specification from bytes
petstore, _ := ioutil.ReadFile("test_specs/petstorev3.json")

// create a new document from specification bytes 
// (ignore errors for the same of the example)
document, _ := NewDocument(petstore)

// because we know this is a v3 spec, we can build a ready to go model from it 
// (ignoring errors for the example)
v3Model, _ := document.BuildV3Model()

// extract the RequestBody from the 'put' operation under the /pet path
reqBody := h.Paths.PathItems["/pet"].Put.RequestBody

// dropdown to the low-level API for RequestBody
lowReqBody := reqBody.GoLow() 

// print out the value, the line it appears on and the 
// start columns for the key and value of the nodes.
fmt.Printf("value is %s, the value is on line %d, " + 
    "starting column %d, the key is on line %d, column %d", 
    reqBody.Description, 
    lowReqBody.Description.ValueNode.Line, 
    lowReqBody.Description.ValueNode.Column,
    lowReqBody.Description.KeyNode.Line, 
    lowReqBody.KeyNode.Column)
```

The library heavily depends on the fantastic (yet hard to get used to) [yaml.Node API](https://pkg.go.dev/gopkg.in/yaml.v3#Node).
This is what is exposed by the `GoLow` API. It does not matter if the input material is JSON or YAML, the yaml.Node API
creates a great way to navigate the AST of the document.


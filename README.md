<p align="center">
	<img src="libopenapi-logo.png" alt="libopenapi" height="300px" width="450px"/>
</p>

# libopenapi - enterprise grade OpenAPI tools for golang.


![Pipeline](https://github.com/pb33f/libopenapi/workflows/Build/badge.svg)
[![GoReportCard](https://goreportcard.com/badge/github.com/pb33f/libopenapi)](https://goreportcard.com/report/github.com/pb33f/libopenapi)
[![codecov](https://codecov.io/gh/pb33f/libopenapi/branch/main/graph/badge.svg?)](https://codecov.io/gh/pb33f/libopenapi)
[![discord](https://img.shields.io/discord/923258363540815912)](https://discord.gg/x7VACVuEGP)
[![Docs](https://img.shields.io/badge/godoc-reference-5fafd7)](https://pkg.go.dev/github.com/pb33f/libopenapi)

libopenapi has full support for Swagger (OpenAPI 2), OpenAPI 3, and OpenAPI 3.1. It can handle the largest and most
complex specifications you can think of.

---

## Sponsors & users
If your company is using `libopenapi`, please considering [supporting this project](https://github.com/sponsors/daveshanley), 
like our _very kind_ sponsors:

<p align="center">
	<a href="//www.speakeasyapi.dev"><img src=".github/sponsors/speakeasy.png" alt="Speakeasy" height="100px"/></a>
    <br/>
    <a href="//www.speakeasyapi.dev">Speakeasy</a>
</p>

`libopenapi` is pretty new, so our list of notable projects that depend on `libopenapi` is small (let me know if you'd like to add your project)

- [github.com/danielgtaylor/restish](https://github.com/danielgtaylor/restish) - "Restish is a CLI for interacting with REST-ish HTTP APIs"
- [github.com/daveshanley/vacuum](https://github.com/daveshanley/vacuum) - "The world's fastest and most scalable OpenAPI/Swagger linter/quality tool"

---

## Come chat with us

Need help? Have a question? Want to share your work? [Join our discord](https://discord.gg/x7VACVuEGP) and
come say hi!

## Contents

- [Installing libopenapi](#installing)
- [Load an OpenAPI 3.1 or 3.0 specification into a model](#load-an-openapi-spec-into-a-model)
- [Load a Swagger Spec into a model](#load-a-swagger-spec-into-a-model)
- [Discover what changed](#discover-what-changed)
- [Loading complex types using extensions](#loading-extensions-using-complex-types)
- [Creating an index of an OpenAPI specification](#creating-an-index-of-an-openapi-specification)
- [Resolving an OpenAPI specification](#resolving-an-openapi-specification)
- [Checking for circular errors without resolving](#checking-for-circular-errors-without-resolving)
- [Extracting circular refs and resolving errors when building a document](#extracting-circular-refs-and-resolving-errors-when-building-a-document)
- [Mutating the model](#mutating-the-model)

> **Read the docs at [https://pkg.go.dev/github.com/pb33f/libopenapi](https://pkg.go.dev/github.com/pb33f/libopenapi)**
> 
---

## Introduction - Why?

There is already a really great OpenAPI library for golang, it's called [kin-openapi](https://github.com/getkin/kin-openapi).

### Why does `libopenapi` exist?

[kin-openapi](https://github.com/getkin/kin-openapi) is great, and you should go and use it.

#### If you're still reading, here is why `libopenapi` might be useful.

>  **_kin-openapi missing a few critical features_**... They are so important, this entire toolset was created to address
> those gaps.

When building tooling that needs to analyze OpenAPI specifications at a *low* level, [kin-openapi](https://github.com/getkin/kin-openapi)
**runs out of power** when you need to know the original line numbers and columns, or comments within all keys and values 
in the specification.

All that data is **lost** when the OpenAPI specification is loaded in by [kin-openapi](https://github.com/getkin/kin-openapi). 
Mainly because the library will unmarshal data **directly into structs**, which works great - if you **_don't_** need 
access to the original specification low level details.

### Why not just modify kin-openapi?

It would require a fundamental re-build of the entire library, with a different design to expose the same functionality.

---

## libopenapi retains _everything_.

`libopenapi` has been designed to retain all of that really low-level detail about the AST, line numbers, column numbers, 
comments, original AST structure - everything you could need.

`libopenapi` has a **porcelain** (high-level) and a **plumbing** (low-level) API. Every high level struct, has the 
ability to `GoLow()` and dive from the high-level model, down to the low-level model and look-up any detail about the 
underlying raw data backing that model.

This library exists because this very need existed inside [VMware](https://vmware.com). The company built an internal 
version of `libopenapi`, which isn't something that can be released as it's customized for VMware (and it's incomplete).

`libopenapi` is the result of years of learning and battle testing OpenAPI in golang. This library represents what would
have been created, if we knew then - what we know now.

## what-changed: a complete diff engine

Compare Swagger and OpenAPI documents for all changes made across the specification. Every detail is checked across
all versions of OpenAPI and returns a simple to navigate report of every single change made. 

Everything modified, added or removed is reported, complete with original/new values, line numbers and columns. 

> Need to know which **line**, or **column** number a key or value for something is? **`libopenapi` has you covered**.

## Comes with an Index and a Resolver

Want a lightning fast way to look up any element in an OpenAPI swagger spec? **`libopenapi` has you covered**.

Need a way to 'resolve' OpenAPI documents that are exploded out across multiple files, remotely or locally? 
**`libopenapi` has you covered**.

---
### Quick-start tutorial

👀 **Get rolling fast using `libopenapi` with the 
[Parsing OpenAPI files using go](https://quobix.com/articles/parsing-openapi-using-go/)** guide 👀

> Read the full docs at [https://pkg.go.dev](https://pkg.go.dev/github.com/pb33f/libopenapi)

---

## Installing
Grab the latest release of **libopenapi**

```
go get github.com/pb33f/libopenapi
```

## Load an OpenAPI spec into a model

```go
// import the library
import "github.com/pb33f/libopenapi"

func readSpec() {
    
    // load an OpenAPI 3 specification from bytes
    petstore, _ := ioutil.ReadFile("test_specs/petstorev3.json")
    
    // create a new document from specification bytes
    document, err := libopenapi.NewDocument(petstore)
    
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
        panic(fmt.Sprintf("cannot create v3 model from document: %d errors reported", 
            len(errors)))
    }
    
    // get a count of the number of paths and schemas.
    paths := len(v3Model.Model.Paths.PathItems)
    schemas := len(v3Model.Model.Components.Schemas)
    
    // print the number of paths and schemas in the document
    fmt.Printf("There are %d paths and %d schemas in the document", paths, schemas)
}
```

This will print: 

```
There are 13 paths and 8 schemas in the document
```


## Load a Swagger spec into a model
```go
// import the library
import "github.com/pb33f/libopenapi"

func readSpec() {

    // load a Swagger specification from bytes
    petstore, _ := ioutil.ReadFile("test_specs/petstorev2.json")
    
    // create a new document from specification bytes
    document, err := libopenapi.NewDocument(petstore)
    
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
        panic(fmt.Sprintf("cannot create v3 model from document: %d errors reported", 
            len(errors)))
    }
    
    // get a count of the number of paths and schemas.
    paths := len(v2Model.Model.Paths.PathItems)
    schemas := len(v2Model.Model.Definitions.Definitions)
    
    // print the number of paths and schemas in the document
    fmt.Printf("There are %d paths and %d schemas in the document", paths, schemas)
}
```

This will print: 

```
There are 14 paths and 6 schemas in the document
```

### Dropping down from the high-level API to the low-level one

This example shows how after loading an OpenAPI spec into a document, navigating to an Operation is pretty simple. 
It then shows how to _drop-down_ (using `GoLow())` to the low-level API and query the line and start 
column of the RequestBody description.

```go
// load an OpenAPI 3 specification from bytes
petstore, _ := ioutil.ReadFile("test_specs/petstorev3.json")

// create a new document from specification bytes 
// (ignore errors for the same of the example)
document, _ := libopenapi.NewDocument(petstore)

// because we know this is a v3 spec, we can build a ready to go model from it 
// (ignoring errors for the example)
v3Model, _ := document.BuildV3Model()

// extract the RequestBody from the 'put' operation under the /pet path
reqBody := document.Paths.PathItems["/pet"].Put.RequestBody

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
---

## Discover what changed

libopenapi comes with a complete **diff engine**

Want to know what changed between two different OpenAPI or Swagger specifications? libopenapi makes this super, super easy.

```go
// How to compare two different OpenAPI specifications for changes between them.

// load an original OpenAPI 3 specification from bytes
burgerShopOriginal, _ := ioutil.ReadFile("test_specs/burgershop.openapi.yaml")

// load an **updated** OpenAPI 3 specification from bytes
burgerShopUpdated, _ := ioutil.ReadFile("test_specs/burgershop.openapi-modified.yaml")

// create a new document from original specification bytes
originalDoc, err := NewDocument(burgerShopOriginal)

// if anything went wrong, an error is thrown
if err != nil {
    panic(fmt.Sprintf("cannot create new document: %e", err))
}

// create a new document from updated specification bytes
updatedDoc, err := NewDocument(burgerShopUpdated)

// if anything went wrong, an error is thrown
if err != nil {
    panic(fmt.Sprintf("cannot create new document: %e", err))
}

// Compare documents for all changes made
documentChanges, errs := CompareDocuments(originalDoc, updatedDoc)
    
// If anything went wrong when building models for documents.
if len(errs) > 0 {
    for i := range errs {
        fmt.Printf("error: %e\n", errs[i])
    }
    panic(fmt.Sprintf("cannot compare documents: %d errors reported", len(errs)))
}

// Extract SchemaChanges from components changes.
schemaChanges := documentChanges.ComponentsChanges.SchemaChanges

// Print out some interesting stats about the OpenAPI document changes.
fmt.Printf("There are %d changes, of which %d are breaking. %v schemas have changes.",
    documentChanges.TotalChanges(), documentChanges.TotalBreakingChanges(), len(schemaChanges))

```

Every change can be explored and navigated just like you would use the high or low level models.

---

## Loading extensions using complex types

If you're using extensions with complex types (rather that just simple strings and primitives), then there is a good
chance you're going to want some simple way to marshal extensions into those structs.

Since version v0.3.2 There is a new method to make this simple available in the `high` package within the `datamodel` 
package. It's called `UnpackExtensions`

It's a generic function that requires the custom type and the **low** model type of the object that contains the extensions
you'd like to unpack.

Here is an example of complex types being extracted easily from OpenAPI extensions.

```go

import (
    "github.com/pb33f/libopenapi"
    high "github.com/pb33f/libopenapi/datamodel/high"
    lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
    lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// define an example struct representing a cake
// super important to remember to use hints/meta-data to map properties correctly.
type cake struct {
    Candles               int    `yaml:"candles"`
    Frosting              string `yaml:"frosting"`
    Some_Strange_Var_Name string `yaml:"someStrangeVarName"`
}

// define a struct that holds a map of cake pointers.
type cakes struct {
    Description string
    Cakes       map[string]*cake
}

// define a struct representing a burger
type burger struct {
    Sauce string
    Patty string
}

// define a struct that holds a map of cake pointers
type burgers struct {
    Description string
    Burgers     map[string]*burger
}

func main() {

    // create a specification with a schema and parameter that use complex 
    // custom cakes and burgers extensions.
    spec := `openapi: "3.1"
    components:
    schemas:
    SchemaOne:
      description: "Some schema with custom complex extensions"
      x-custom-cakes:
        description: some cakes
        cakes:
          someCake:
            candles: 10
            frosting: blue
            someStrangeVarName: mapping is required to extract these.
          anotherCake:
            candles: 1
            frosting: green
    parameters:
    ParameterOne:
      description: "Some parameter also using complex extensions"
      x-custom-burgers:
        description: some burgers
        burgers:
          someBurger:
            sauce: ketchup
            patty: meat 
          anotherBurger:
            sauce: mayo
            patty: lamb`
    
    // create a new document from specification bytes
    doc, err := libopenapi.NewDocument([]byte(spec))
    
    // if anything went wrong, an error is thrown
    if err != nil {
        panic(fmt.Sprintf("cannot create new document: %e", err))
    }
    
    // build a v3 model.
    docModel, errs := doc.BuildV3Model()
    
    // if anything went wrong building, indexing and resolving the model, errors are thrown.
    if errs != nil {
        panic(fmt.Sprintf("cannot create new document: %e", errs))
    }
    
    // get a reference to SchemaOne and ParameterOne
    schemaOne := docModel.Model.Components.Schemas["SchemaOne"].Schema()
    parameterOne := docModel.Model.Components.Parameters["ParameterOne"]
    
    // unpack schemaOne extensions into complex `cakes` type
    schemaOneExtensions, schemaUnpackErrors := high.UnpackExtensions[cakes, *lowbase.Schema](schemaOne)
    if schemaUnpackErrors != nil {
        panic(fmt.Sprintf("cannot unpack schema extensions: %e", err))
    }
    
    // unpack parameterOne into complex `burgers` type
    parameterOneExtensions, paramUnpackErrors := high.UnpackExtensions[burgers, *lowv3.Parameter](parameterOne)
    if paramUnpackErrors != nil {
        panic(fmt.Sprintf("cannot unpack parameter extensions: %e", err))
    }
    
    // extract extension by name for schemaOne
    customCakes := schemaOneExtensions["x-custom-cakes"]
    
    // extract extension by name for schemaOne
    customBurgers := parameterOneExtensions["x-custom-burgers"]
    
    // print out schemaOne complex extension details.
    fmt.Printf("schemaOne 'x-custom-cakes' (%s) has %d cakes, 'someCake' has %d candles and %s frosting\n",
        customCakes.Description,
        len(customCakes.Cakes),
        customCakes.Cakes["someCake"].Candles,
        customCakes.Cakes["someCake"].Frosting,
    )
    
    // print out parameterOne complex extension details.
    fmt.Printf("parameterOne 'x-custom-burgers' (%s) has %d burgers, 'anotherBurger' has %s sauce and a %s patty\n",
        customBurgers.Description,
        len(customBurgers.Burgers),
        customBurgers.Burgers["anotherBurger"].Sauce,
        customBurgers.Burgers["anotherBurger"].Patty,
    )
}
```

This will output:

```text
schemaOne 'x-custom-cakes' (some cakes) has 2 cakes, 'someCake' has 10 candles and blue frosting
parameterOne 'x-custom-burgers' (some burgers) has 2 burgers, 'anotherBurger' has mayo sauce and a lamb patty
```
---

## Creating an index of an OpenAPI Specification

An index is really useful when a map of an OpenAPI spec is needed. Knowing where all the references are and where
they point, is very useful when resolving specifications, or just looking things up. 

### Creating an index from the Stripe OpenAPI Spec

```go
// define a rootNode to hold our raw stripe spec AST.
var rootNode yaml.Node

// load in the stripe OpenAPI specification into bytes (it's pretty meaty)
stripeSpec, _ := ioutil.ReadFile("test_specs/stripe.yaml")

// unmarshal spec into our rootNode
yaml.Unmarshal(stripeSpec, &rootNode)

// create a new specification index.
index := index.NewSpecIndex(&rootNode)

// print out some statistics
fmt.Printf("There are %d references\n"+
    "%d paths\n"+
    "%d operations\n"+
    "%d schemas\n"+
    "%d enums\n"+
    "%d polymorphic references",
    len(index.GetAllCombinedReferences()),
    len(index.GetAllPaths()),
    index.GetOperationCount(),
    len(index.GetAllSchemas()),
    len(index.GetAllEnums()),
    len(index.GetPolyOneOfReferences())+len(index.GetPolyAnyOfReferences()))
```

## Resolving an OpenAPI Specification

When creating an index, the raw AST that uses [yaml.Node](https://pkg.go.dev/gopkg.in/yaml.v3#Node) is preserved 
when looking up local, file-based and remote references. This means that if required, the spec can be 'resolved' 
and all the reference nodes will be replaced with the actual data they reference.

What this looks like from a spec perspective.

If the specification looks like this:

```yaml
paths:
  "/some/path/to/a/thing":
    get:
      responses:
        "200":
          $ref: '#/components/schemas/MySchema'
components:
  schemas:
   MySchema:
     type: string
     description: This is my schema that is great!
```

Will become this (as represented by the root [yaml.Node](https://pkg.go.dev/gopkg.in/yaml.v3#Node)

```yaml
paths:
  "/some/path/to/a/thing":
    get:
      responses:
        "200":
          type: string
          description: This is my schema that is great!
components:
  schemas:
   MySchema:
     type: string
     description: This is my schema that is great!
```
> This is not a valid spec, it's just design to illustrate how resolving works.

The reference has been 'resolved', so when reading the raw AST, there is no lookup required anymore.

### Resolving Example:

Using the Stripe API as an example, we can resolve all references, and then count how many circular reference issues
were found.

```go
import (
    "github.com/pb33f/libopenapi/index"
    "github.com/pb33f/libopenapi/resolver"
)

// create a yaml.Node reference as a root node.
var rootNode yaml.Node

//  load in the Stripe OpenAPI spec (lots of polymorphic complexity in here)
stripeBytes, _ := ioutil.ReadFile("../test_specs/stripe.yaml")

// unmarshal bytes into our rootNode.
_ = yaml.Unmarshal(stripeBytes, &rootNode)

// create a new spec index (resolver depends on it)
index := index.NewSpecIndex(&rootNode)

// create a new resolver using the index.
resolver := resolver.NewResolver(index)

// resolve the document, if there are circular reference errors, they are returned/
// WARNING: this is a destructive action and the rootNode will be 
// PERMANENTLY altered and cannot be unresolved
circularErrors := resolver.Resolve()

// The Stripe API has a bunch of circular reference problems, 
// mainly from polymorphism. So let's print them out.
fmt.Printf("There are %d circular reference errors, " +
    "%d of them are polymorphic errors, %d are not",
    len(circularErrors), 
    len(resolver.GetPolymorphicCircularErrors()), 
    len(resolver.GetNonPolymorphicCircularErrors()))
```

This will output: 

`There are 21 circular reference errors, 19 of them are polymorphic errors, 2 are not`

> Important to remember: Resolving is **destructive** and will permanently change the tree, it cannot be un-resolved.

### Checking for circular errors without resolving

Resolving is destructive, the original reference nodes are gone and all replaced, so how do we check for circular references
in a non-destructive way? Instead of calling `Resolve()`, we can call `CheckForCircularReferences()` instead.

The same code as `Resolve()` executes, except the tree is **not actually resolved**, _nothing_ changes and _no destruction_
occurs. A handy way to perform circular reference analysis on the specification, without permanently altering it.

```go
import (
    "github.com/pb33f/libopenapi/index" 
    "github.com/pb33f/libopenapi/resolver"
)

// create a yaml.Node reference as a root node.
var rootNode yaml.Node

//  load in the Stripe OpenAPI spec (lots of polymorphic complexity in here)
stripeBytes, _ := ioutil.ReadFile("../test_specs/stripe.yaml")

// unmarshal bytes into our rootNode.
_ = yaml.Unmarshal(stripeBytes, &rootNode)

// create a new spec index (resolver depends on it)
index := index.NewSpecIndex(&rootNode)

// create a new resolver using the index.
resolver := resolver.NewResolver(index)

// extract circular reference errors without any changes to the original tree.
circularErrors := resolver.CheckForCircularReferences()

// The Stripe API has a bunch of circular reference problems, 
// mainly from polymorphism. So let's print them out.
fmt.Printf("There are %d circular reference errors, " +
    "%d of them are polymorphic errors, %d are not",
    len(circularErrors),
    len(resolver.GetPolymorphicCircularErrors()),
    len(resolver.GetNonPolymorphicCircularErrors()))
```

### Extracting circular refs and resolving errors when building a document

To avoid having to create an index and a resolver each time you want to both create
a document and resolve it / check for errors, don't worry, circular references are checked
automatically and are available in the returned `[]errors` which building a document.

The errors returned by the slice are pointers to `*resolver.ResolvingError` which contains
rich details about the issue, where it was found and the journey it took to get there.

Here is an example:

```go
// create a specification with an obvious and deliberate circular reference 
spec := `
openapi: "3.1"
components:
  schemas:
    One:
      description: "test one"
      properties:
        things:
          "$ref": "#/components/schemas/Two"
    Two:
      description: "test two"
      properties:
        testThing:
          "$ref": "#/components/schemas/One"
`
// create a new document from specification bytes
doc, err := NewDocument([]byte(spec))

// if anything went wrong, an error is thrown
if err != nil {
    panic(fmt.Sprintf("cannot create new document: %e", err))
}
_, errs := doc.BuildV3Model()

// extract resolving error
resolvingError := errs[0]

// resolving error is a pointer to *resolver.ResolvingError
// which provides access to rich details about the error.
circularReference := resolvingError.(*resolver.ResolvingError).CircularReference

// capture the journey with all details
var buf strings.Builder
for n := range circularReference.Journey {

    // add the full definition name to the journey.
    buf.WriteString(circularReference.Journey[n].Definition)
    if n < len(circularReference.Journey)-1 {
        buf.WriteString(" -> ")
    }
}

// print out the journey and the loop point.
fmt.Printf("Journey: %s\n", buf.String())
fmt.Printf("Loop Point: %s", circularReference.LoopPoint.Definition)
```

Will output:

```text
Journey: #/components/schemas/Two -> #/components/schemas/One -> #/components/schemas/Two
Loop Point: #/components/schemas/Two
```

---


## Mutating the model

Having a read-only model is great, but what about when we want to modify the model and mutate values, or even add new
content to the model? What if we also want to save that output as an updated specification - but we don't want to
jumble up the original ordering of the source.

### marshaling and unmarshalling to and from structs into JSON/YAML is not ideal.

When we straight up use `json.Marshal` or `yaml.Marshal` to send structs to be rendered into the desired format, there
is no guarantee as to the order in which each component will be rendered. This works great if...

- We don't care about the spec being randomly ordered.
- We don't care about code-reviews.
- We don't actually care about this very much.

### But if we do care...

Then libopenpi provides a way to mutate the model, that keeps the original [yaml.Node API](https://pkg.go.dev/gopkg.in/yaml.v3#Node)
tree in-tact. It allows us to make changes to values in place, and serialize back to JSON or YAML without any changes to
other content order.

```go
// create very small, and useless spec that does nothing useful, except showcase this feature.
spec := `
openapi: 3.1.0
info:
  title: This is a title
  contact:
    name: Some Person
    email: some@emailaddress.com
  license:
    url: http://some-place-on-the-internet.com/license
`
// create a new document from specification bytes
document, err := libopenapi.NewDocument([]byte(spec))

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

// mutate the title, to do this we currently need to drop down to the low-level API.
v3Model.Model.GoLow().Info.Value.Title.Mutate("A new title for a useless spec")

// mutate the email address in the contact object.
v3Model.Model.GoLow().Info.Value.Contact.Value.Email.Mutate("buckaroo@pb33f.io")

// mutate the name in the contact object.
v3Model.Model.GoLow().Info.Value.Contact.Value.Name.Mutate("Buckaroo")

// mutate the URL for the license object.
v3Model.Model.GoLow().Info.Value.License.Value.URL.Mutate("https://pb33f.io/license")

// serialize the document back into the original YAML or JSON
mutatedSpec, serialError := document.Serialize()

// if something went wrong serializing
if serialError != nil {
    panic(fmt.Sprintf("cannot serialize document: %e", serialError))
}

// print our modified spec!
fmt.Println(string(mutatedSpec))
```

Which will output:

```yaml
openapi: 3.1.0
info:
    title: A new title for a useless spec
    contact:
         name: Buckaroo
         email: buckaroo@pb33f.io
    license:
         url: https://pb33f.io/license

```
> It's worth noting that the original line numbers and column numbers **won't be respected** when calling `Serialize()`,
> A new `Document` needs to be created from that raw YAML to continue processing after serialization.

> **Read the full docs at [https://pkg.go.dev](https://pkg.go.dev/github.com/pb33f/libopenapi)**

---

The library heavily depends on the fantastic (yet hard to get used to) [yaml.Node API](https://pkg.go.dev/gopkg.in/yaml.v3#Node).
This is what is exposed by the `GoLow` API. 

> It does not matter if the input material is JSON or YAML, the [yaml.Node API](https://pkg.go.dev/gopkg.in/yaml.v3#Node) supports both and 
> creates a great way to navigate the AST of the document.

Logo gopher is modified, originally from [egonelbre](https://github.com/egonelbre/gophers)

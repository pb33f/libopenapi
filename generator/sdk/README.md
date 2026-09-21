# SDK generation

`generator/sdk` prepares client contracts from OpenAPI and finite workflows from
Arazzo. Language emitters consume that prepared contract. Generated clients do
not import libopenapi or parse specifications at runtime.

The first emitter is `generator/sdk/gosdk`. It generates:

- reachable models through `generator/golang`;
- one shared HTTP client and security runtime;
- resource-oriented methods such as `client.Widgets.List`;
- typed success values, declared API errors, raw unknown errors, status and
  response headers;
- a narrow generated helper for supported finite Arazzo workflows.

## Generate a Go client

```go
contract, err := sdk.Prepare(openAPIDocument, sdk.PrepareOptions{
    Operations: []string{"listWidgets", "getWidget"},
    Names: map[string]sdk.OperationName{
        "listWidgets": {Resource: "Widgets", Method: "List"},
        "getWidget":  {Resource: "Widgets", Method: "Get"},
    },
})
if err != nil {
    return err
}

result, err := gosdk.GenerateContract(contract, gosdk.Options{
    PackageName: "exampleapi",
})
```

`result.Files` is deterministic and caller-owned. The caller decides where to
write it. `result.Diagnostics` contains notable model-generation decisions.
Unsupported selected wire behavior returns an error naming the operation and
feature; the emitter does not silently drop parameters, bodies, responses, or
security requirements.

## Generated runtime

The generated client accepts an absolute server URL, an injected `HTTPDoer`,
named credentials, default headers, a response-size bound, and a timeout for the
default `http.Client`. Configuration is copied during construction and resource
methods are safe to share between goroutines. Per-call request options operate
on a new request.

Security keeps OpenAPI semantics: requirement objects are alternatives (OR),
while every scheme inside one object is required (AND). Bearer, Basic, API-key,
OAuth/OpenID bearer-token, and mutual-TLS hooks are represented as named
credentials. The default redirect policy returns a cross-origin redirect
response without following it. This also prevents 307 and 308 redirects from
replaying request bodies to another origin. A caller-supplied HTTP client owns
its redirect and TLS policy.

The initial wire surface supports JSON request and response bodies, scalar and
array path/query/header parameters using their standard styles, explicit 2xx
statuses, declared JSON errors, cancellation, and bounded response buffering.
Cookie parameters, object/deep-object parameters, `allowReserved`, form and
multipart bodies, binary downloads, and streaming produce generation errors
until their dedicated runtime paths land.

Optional nullable generated model fields use an opt-in double-pointer shape:

- `nil` outer pointer: omit the field;
- non-nil outer pointer containing `nil`: encode JSON `null`;
- two non-nil pointers: encode the value, including its zero value.

The generated runtime provides `Null[T]` and `NullableValue[T]` helpers.

## Finite Arazzo workflows

`sdk.PrepareArazzo` currently accepts one operation step with direct
`$inputs.<name>` JSON payload bindings and simple status-code success criteria.
The Go emitter turns it into a typed client method using the same generated
resource operation. Unsupported workflow control flow is an error at generation
time.

The finite subset is deliberately explicit: one operation step, direct input
bindings for an object request body, and simple equality checks against HTTP
status codes. Runtime orchestration and long-running observation remain
consumer concerns rather than generated control flow.

## Ownership and next work

OpenAPI remains the HTTP and schema contract. Arazzo remains the finite workflow
contract. Stable runtime code is emitted once per SDK from standard-library
templates; templates receive prepared values and contain no schema traversal.

The next Go increments are form/multipart, binary and streaming responses,
explicit pagination mappings, and consumer observation adapters. TypeScript
should reuse the prepared contract and behavior fixtures after the Go surface is
proven. External SDK repositories and publication remain consumer concerns.

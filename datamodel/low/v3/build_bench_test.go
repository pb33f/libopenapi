// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"go.yaml.in/yaml/v4"
)

func benchmarkMediaTypeRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `schema:
  type: object
  properties:
    name:
      type: string
example:
  nested:
    value:
      - hello
      - world
examples:
  what:
    value:
      why: there
  where:
    value:
      here: now
encoding:
  chicken:
    explode: true
x-rock:
  and: roll`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark media type: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark media type: empty root")
	}
	return root.Content[0]
}

func benchmarkParameterRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `description: michelle, meddy and maddy
required: true
deprecated: false
name: happy
in: path
allowEmptyValue: false
style: beautiful
explode: true
allowReserved: true
schema:
  type: object
  description: my triple M, my loves
  properties:
    michelle:
      type: string
    meddy:
      type: string
    maddy:
      type: string
example:
  michelle: my love.
  maddy: my champion.
  meddy: my song.
content:
  family/love:
    schema:
      type: string
      description: family love.
examples:
  family:
    value:
      michelle: my love.
      maddy: my champion.
      meddy: my song.
x-family-love:
  strong: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark parameter: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark parameter: empty root")
	}
	return root.Content[0]
}

func benchmarkOperationRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `tags:
  - create
  - pizza
summary: create a pizza
description: takes ingredients and produces pizza
externalDocs:
  description: docs
  url: https://example.com/docs
operationId: createPizza
parameters:
  - name: style
    in: query
    schema:
      type: string
requestBody:
  description: incoming pizza
  content:
    application/json:
      schema:
        type: object
        properties:
          name:
            type: string
responses:
  "200":
    description: ok
callbacks:
  status:
    "{$request.body#/callbackUrl}":
      post:
        responses:
          "200":
            description: ok
security:
  - apiKey: []
servers:
  - url: https://api.example.com
x-pizza:
  hot: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark operation: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark operation: empty root")
	}
	return root.Content[0]
}

func benchmarkServerRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `name: Production
url: https://{region}.api.example.com/{version}
description: regional server
variables:
  region:
    default: us-east-1
    description: deployment region
    enum:
      - us-east-1
      - eu-west-1
  version:
    default: v1
    description: api version
    enum:
      - v1
      - v2
x-server:
  blue: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark server: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark server: empty root")
	}
	return root.Content[0]
}

func benchmarkPathItemRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `summary: pizza path
description: handles pizza endpoints
parameters:
  - name: orgId
    in: path
    required: true
    schema:
      type: string
servers:
  - url: https://api.example.com
get:
  summary: get a pizza
  operationId: getPizza
  responses:
    "200":
      description: ok
post:
  summary: create a pizza
  operationId: createPizza
  requestBody:
    content:
      application/json:
        schema:
          type: object
          properties:
            name:
              type: string
  responses:
    "201":
      description: created
x-path:
  fast: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark path item: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark path item: empty root")
	}
	return root.Content[0]
}

func benchmarkPathsRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `"/pizza":
  get:
    summary: get pizza
    responses:
      "200":
        description: ok
  post:
    summary: create pizza
    requestBody:
      content:
        application/json:
          schema:
            type: object
    responses:
      "201":
        description: created
"/burger":
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
  get:
    summary: get burger
    responses:
      "200":
        description: ok
x-menu:
  hot: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark paths: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark paths: empty root")
	}
	return root.Content[0]
}

func benchmarkComponentsRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `schemas:
  Pet:
    type: object
    properties:
      id:
        type: string
      name:
        type: string
responses:
  Ok:
    description: ok
parameters:
  petId:
    name: petId
    in: path
    required: true
    schema:
      type: string
examples:
  PetExample:
    value:
      id: 1
      name: dog
requestBodies:
  PetBody:
    content:
      application/json:
        schema:
          $ref: '#/schemas/Pet'
headers:
  RateLimit:
    description: rate
securitySchemes:
  ApiKey:
    type: apiKey
    in: header
    name: X-API-Key
links:
  PetLink:
    operationId: getPet
callbacks:
  PetCallback:
    "{$request.body#/callbackUrl}":
      post:
        responses:
          "200":
            description: ok
pathItems:
  /pets:
    get:
      responses:
        "200":
          description: ok
mediaTypes:
  json:
    schema:
      type: string
x-components:
  hot: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark components: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark components: empty root")
	}
	return root.Content[0]
}

func benchmarkResponseRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `summary: success
description: some response
headers:
  rate:
    description: rate header
content:
  application/json:
    schema:
      type: object
      properties:
        message:
          type: string
links:
  follow:
    operationId: getThing
x-response:
  good: yes`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark response: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark response: empty root")
	}
	return root.Content[0]
}

func benchmarkRequestBodyRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `description: request body
required: true
content:
  application/json:
    schema:
      type: object
      properties:
        name:
          type: string
    example:
      name: pizza
x-request:
  hot: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark request body: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark request body: empty root")
	}
	return root.Content[0]
}

func benchmarkHeaderRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `description: header
required: true
deprecated: false
allowEmptyValue: false
style: simple
explode: true
allowReserved: false
schema:
  type: object
  properties:
    name:
      type: string
example:
  name: pizza
examples:
  sample:
    value:
      name: pie
content:
  application/json:
    schema:
      type: string
x-header:
  bright: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark header: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark header: empty root")
	}
	return root.Content[0]
}

func benchmarkResponsesRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `"200":
  summary: success
  description: ok
  headers:
    rate:
      description: rate limit
  content:
    application/json:
      schema:
        type: object
        properties:
          message:
            type: string
  links:
    next:
      operationId: nextThing
default:
  description: fallback
x-responses:
  hot: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark responses: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark responses: empty root")
	}
	return root.Content[0]
}

func benchmarkCallbackRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `'{$request.query.queryUrl}':
  post:
    requestBody:
      description: callback payload
      content:
        application/json:
          schema:
            type: object
            properties:
              message:
                type: string
    responses:
      "200":
        description: ok
x-callback:
  warm: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark callback: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark callback: empty root")
	}
	return root.Content[0]
}

func benchmarkOAuthFlowsRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `implicit:
  authorizationUrl: https://auth.example.com/authorize
  tokenUrl: https://auth.example.com/token
  refreshUrl: https://auth.example.com/refresh
  scopes:
    read: read things
    write: write things
authorizationCode:
  authorizationUrl: https://auth.example.com/code
  tokenUrl: https://auth.example.com/token
  scopes:
    admin: admin things
device:
  tokenUrl: https://auth.example.com/device
  scopes:
    device: device things
x-flows:
  warm: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark oauth flows: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark oauth flows: empty root")
	}
	return root.Content[0]
}

func benchmarkSecuritySchemeRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `type: oauth2
description: auth
scheme: bearer
bearerFormat: jwt
openIdConnectUrl: https://auth.example.com/openid
oauth2MetadataUrl: https://auth.example.com/.well-known/oauth-authorization-server
deprecated: false
flows:
  implicit:
    authorizationUrl: https://auth.example.com/authorize
    tokenUrl: https://auth.example.com/token
    scopes:
      read: read things
  device:
    tokenUrl: https://auth.example.com/device
    scopes:
      device: device things
x-security:
  strict: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark security scheme: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark security scheme: empty root")
	}
	return root.Content[0]
}

func benchmarkLinkRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `operationRef: '#/paths/~1pets/get'
operationId: getPet
parameters:
  petId: $response.body#/id
  traceId: $response.header.X-Trace
requestBody: $request.body#/payload
description: follow the pet
server:
  url: https://api.example.com
  variables:
    version:
      default: v1
x-link:
  bright: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark link: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark link: empty root")
	}
	return root.Content[0]
}

func benchmarkEncodingRootNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `contentType: application/json
headers:
  x-rate:
    description: rate header
    required: true
    schema:
      type: integer
  x-mode:
    description: mode header
    schema:
      type: string
style: form
explode: true
allowReserved: true`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark encoding: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark encoding: empty root")
	}
	return root.Content[0]
}

func BenchmarkMediaType_Build(b *testing.B) {
	rootNode := benchmarkMediaTypeRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var mt MediaType
		if err := low.BuildModel(rootNode, &mt); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := mt.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("media type build failed: %v", err)
		}
	}
}

func BenchmarkParameter_Build(b *testing.B) {
	rootNode := benchmarkParameterRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var p Parameter
		if err := low.BuildModel(rootNode, &p); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := p.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("parameter build failed: %v", err)
		}
	}
}

func BenchmarkOperation_Build(b *testing.B) {
	rootNode := benchmarkOperationRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var o Operation
		if err := low.BuildModel(rootNode, &o); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := o.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("operation build failed: %v", err)
		}
	}
}

func BenchmarkServer_Build(b *testing.B) {
	rootNode := benchmarkServerRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var s Server
		if err := low.BuildModel(rootNode, &s); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := s.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("server build failed: %v", err)
		}
	}
}

func BenchmarkPathItem_Build(b *testing.B) {
	rootNode := benchmarkPathItemRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var p PathItem
		if err := low.BuildModel(rootNode, &p); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := p.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("path item build failed: %v", err)
		}
	}
}

func BenchmarkPaths_Build(b *testing.B) {
	rootNode := benchmarkPathsRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var p Paths
		if err := low.BuildModel(rootNode, &p); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := p.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("paths build failed: %v", err)
		}
	}
}

func BenchmarkComponents_Build(b *testing.B) {
	rootNode := benchmarkComponentsRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var c Components
		if err := low.BuildModel(rootNode, &c); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := c.Build(ctx, rootNode, idx); err != nil {
			b.Fatalf("components build failed: %v", err)
		}
	}
}

func BenchmarkResponse_Build(b *testing.B) {
	rootNode := benchmarkResponseRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var r Response
		if err := low.BuildModel(rootNode, &r); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := r.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("response build failed: %v", err)
		}
	}
}

func BenchmarkRequestBody_Build(b *testing.B) {
	rootNode := benchmarkRequestBodyRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var rb RequestBody
		if err := low.BuildModel(rootNode, &rb); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := rb.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("request body build failed: %v", err)
		}
	}
}

func BenchmarkHeader_Build(b *testing.B) {
	rootNode := benchmarkHeaderRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var h Header
		if err := low.BuildModel(rootNode, &h); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := h.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("header build failed: %v", err)
		}
	}
}

func BenchmarkResponses_Build(b *testing.B) {
	rootNode := benchmarkResponsesRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var r Responses
		if err := low.BuildModel(rootNode, &r); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := r.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("responses build failed: %v", err)
		}
	}
}

func BenchmarkCallback_Build(b *testing.B) {
	rootNode := benchmarkCallbackRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var cb Callback
		if err := low.BuildModel(rootNode, &cb); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := cb.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("callback build failed: %v", err)
		}
	}
}

func BenchmarkOAuthFlows_Build(b *testing.B) {
	rootNode := benchmarkOAuthFlowsRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var flows OAuthFlows
		if err := low.BuildModel(rootNode, &flows); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := flows.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("oauth flows build failed: %v", err)
		}
	}
}

func BenchmarkSecurityScheme_Build(b *testing.B) {
	rootNode := benchmarkSecuritySchemeRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var scheme SecurityScheme
		if err := low.BuildModel(rootNode, &scheme); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := scheme.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("security scheme build failed: %v", err)
		}
	}
}

func BenchmarkLink_Build(b *testing.B) {
	rootNode := benchmarkLinkRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var link Link
		if err := low.BuildModel(rootNode, &link); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := link.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("link build failed: %v", err)
		}
	}
}

func BenchmarkEncoding_Build(b *testing.B) {
	rootNode := benchmarkEncodingRootNode(b)
	idx := index.NewSpecIndex(rootNode)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var encoding Encoding
		if err := low.BuildModel(rootNode, &encoding); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
		if err := encoding.Build(ctx, nil, rootNode, idx); err != nil {
			b.Fatalf("encoding build failed: %v", err)
		}
	}
}

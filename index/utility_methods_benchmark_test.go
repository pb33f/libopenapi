// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"runtime"
	"testing"

	"github.com/pkg-base/yaml"
)

// Benchmark buffer pool optimization vs original allocation pattern
func BenchmarkHashNode_BufferPool(b *testing.B) {
	// Complex nested YAML to test deep recursion and multiple buffer reuses
	complexYAML := `
openapi: 3.0.3
info:
  title: Benchmark API
  version: 1.0.0
paths:
  /users/{id}:
    get:
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: User found
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: integer
                  name:
                    type: string
                  email:
                    type: string
                  address:
                    type: object
                    properties:
                      street:
                        type: string
                      city:
                        type: string
                      country:
                        type: string
  /users:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                email:
                  type: string
                address:
                  type: object
                  properties:
                    street:
                      type: string
                    city:
                      type: string
                    country:
                      type: string
      responses:
        '201':
          description: User created
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        email:
          type: string
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(complexYAML), &rootNode)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = HashNode(&rootNode)
	}
}

// Benchmark with multiple concurrent goroutines to test sync.Pool behavior
func BenchmarkHashNode_Concurrent(b *testing.B) {
	complexYAML := `
openapi: 3.0.3
info:
  title: Concurrent Test
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      type: object
                      properties:
                        id: 
                          type: integer
                        value:
                          type: string
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(complexYAML), &rootNode)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = HashNode(&rootNode)
		}
	})
}

// Memory allocation benchmark to measure improvement
func BenchmarkHashNode_MemoryAlloc(b *testing.B) {
	yamlStr := `
test:
  nested:
    deeply:
      nested:
        values:
          - item1: value1
          - item2: value2
          - item3: value3
        more:
          data:
            here:
              and:
                there: everywhere
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yamlStr), &rootNode)
	if err != nil {
		b.Fatal(err)
	}

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HashNode(&rootNode)
	}
	b.StopTimer()

	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "allocs/op")
}

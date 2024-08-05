package json_test

import (
	"testing"

	"github.com/pb33f/libopenapi/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestYAMLNodeToJSON(t *testing.T) {
	y := `root:
  key1: scalar1
  key2: 
    - scalar2
    - subkey1: scalar3
      subkey2:
        - 1
        - 2
    -
      - scalar4
      - scalar5
  key3: true`

	var v yaml.Node

	err := yaml.Unmarshal([]byte(y), &v)
	require.NoError(t, err)

	j, err := json.YAMLNodeToJSON(&v, "  ")
	require.NoError(t, err)

	assert.Equal(t, `{
  "root": {
    "key1": "scalar1",
    "key2": [
      "scalar2",
      {
        "subkey1": "scalar3",
        "subkey2": [
          1,
          2
        ]
      },
      [
        "scalar4",
        "scalar5"
      ]
    ],
    "key3": true
  }
}`, string(j))
}

func TestYAMLNodeToJSON_FromJSON(t *testing.T) {
	j := `{
  "root": {
    "key1": "scalar1",
    "key2": [
      "scalar2",
      {
        "subkey1": "scalar3",
        "subkey2": [
          1,
          2
        ]
      },
      [
        "scalar4",
        "scalar5"
      ]
    ],
    "key3": true
  }
}`

	var v yaml.Node

	err := yaml.Unmarshal([]byte(j), &v)
	require.NoError(t, err)

	o, err := json.YAMLNodeToJSON(&v, "  ")
	require.NoError(t, err)

	assert.Equal(t, j, string(o))
}

func TestYAMLNodeWithAnchorsToJSON(t *testing.T) {
	y := `examples:
  someExample: &someExample
    key1: scalar1
    key2: scalar2
someValue: *someExample`

	var v yaml.Node

	err := yaml.Unmarshal([]byte(y), &v)
	require.NoError(t, err)

	j, err := json.YAMLNodeToJSON(&v, "  ")
	require.NoError(t, err)

	assert.Equal(t, `{
  "examples": {
    "someExample": {
      "key1": "scalar1",
      "key2": "scalar2"
    }
  },
  "someValue": {
    "key1": "scalar1",
    "key2": "scalar2"
  }
}`, string(j))
}

func TestYAMLNodeWithComplexKeysToJSON(t *testing.T) {
	y := `someMapWithComplexKeys:
  {key1: scalar1, key2: scalar2}: {key1: scalar1, key2: scalar2}`

	var v yaml.Node

	err := yaml.Unmarshal([]byte(y), &v)
	require.NoError(t, err)

	j, err := json.YAMLNodeToJSON(&v, "  ")
	require.NoError(t, err)

	assert.Equal(t, `{
  "someMapWithComplexKeys": {
    "{\"key1\":\"scalar1\",\"key2\":\"scalar2\"}": {
      "key1": "scalar1",
      "key2": "scalar2"
    }
  }
}`, string(j))
}

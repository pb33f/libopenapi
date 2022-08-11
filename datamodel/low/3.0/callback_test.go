package v3

import (
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "testing"
)

func TestCallback_Build_Success(t *testing.T) {

    yml := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: string
      responses:
        '200':
          description: callback successfully processed`

    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    var n Callback
    err := BuildModel(rootNode.Content[0], &n)
    assert.NoError(t, err)

    err = n.Build(rootNode.Content[0], nil)
    assert.NoError(t, err)

    assert.Len(t, n.Expression.Value, 1)

}

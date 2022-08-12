package v3

import (
	"github.com/pb33f/libopenapi/index"
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

func TestCallback_Build_Error(t *testing.T) {

	// first we need an index.
	doc := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(doc), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml := `'{$request.query.queryUrl}':
  post:
    $ref: #/does/not/exist/and/invalid`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	var n Callback
	err := BuildModel(rootNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(rootNode.Content[0], idx)
	assert.Error(t, err)

}

func TestCallback_Build_Using_InlineRef(t *testing.T) {

	// first we need an index.
	doc := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(doc), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml := `'{$request.query.queryUrl}':
    post:
      requestBody:
        $ref: '#/components/schemas/Something'
      responses:
        '200':
          description: callback successfully processed`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	var n Callback
	err := BuildModel(rootNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(rootNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Len(t, n.Expression.Value, 1)

	exp := n.FindExpression("{$request.query.queryUrl}")
	assert.NotNil(t, exp.Value)
	assert.NotNil(t, exp.Value.Post.Value)
	assert.Equal(t, "this is something", exp.Value.Post.Value.RequestBody.Value.Description.Value)

}

func TestCallback_Build_Using_Ref(t *testing.T) {

	// first we need an index.
	doc := `paths:
 '/something/here':
   post:
     requestBody:
       $ref: '#/components/requestBodies/rbOne'
     responses:
       "200":
         $ref: '#/components/schemas/AThing'  
components:
  schemas: 
    AThing:
      description: a thing.
      type: string
  requestBodies: 
    rbOne:
      description: a request body.
      type: string`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(doc), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml := `'{$request.query.queryUrl}':
    $ref: '#/paths/~1something~1here'`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	var n Callback
	err := BuildModel(rootNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(rootNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Len(t, n.Expression.Value, 1)

	exp := n.FindExpression("{$request.query.queryUrl}")
	assert.NotNil(t, exp.Value)
	assert.NotNil(t, exp.Value.Post.Value)
	assert.Equal(t, "a request body.", exp.Value.Post.Value.RequestBody.Value.Description.Value)

	okCode := exp.Value.Post.Value.Responses.Value.FindResponseByCode("200")
	assert.Equal(t, "a thing.", okCode.Value.Description.Value)

}

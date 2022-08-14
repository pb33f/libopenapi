package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestEncoding_Build_Success(t *testing.T) {

	yml := `contentType: hot/cakes
headers: 
  ohMyStars:
    description: this is a header
    required: true
    allowEmptyValue: true
allowReserved: true    
explode: true`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Encoding
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "hot/cakes", n.ContentType.Value)
	assert.Equal(t, true, n.AllowReserved.Value)
	assert.Equal(t, true, n.Explode.Value)

	header := n.FindHeader("ohMyStars")
	assert.NotNil(t, header.Value)
	assert.Equal(t, "this is a header", header.Value.Description.Value)
	assert.Equal(t, true, header.Value.Required.Value)
	assert.Equal(t, true, header.Value.AllowEmptyValue.Value)
}

func TestEncoding_Build_Error(t *testing.T) {

	yml := `contentType: hot/cakes
headers: 
  $ref: #/borked`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Encoding
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

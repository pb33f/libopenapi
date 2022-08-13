package v3

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestDiscriminator_FindMappingValue(t *testing.T) {
	yml := `propertyName: freshCakes
mapping:
  something: nothing`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)

	var n Discriminator
	err := BuildModel(&idxNode, &n)
	assert.NoError(t, err)
	assert.Equal(t, "nothing", n.FindMappingValue("something").Value)
	assert.Nil(t, n.FindMappingValue("freshCakes"))

}

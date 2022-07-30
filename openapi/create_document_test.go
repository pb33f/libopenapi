package openapi

import (
    "github.com/stretchr/testify/assert"
    "io/ioutil"
    "testing"
)

func TestCreateDocument_NoData(t *testing.T) {
    doc, err := CreateDocument(nil)
    assert.Nil(t, doc)
    assert.Error(t, err)
}

func TestCreateDocument(t *testing.T) {
    data, aErr := ioutil.ReadFile("../test_specs/burgershop.openapi.yaml")
    assert.NoError(t, aErr)

    doc, err := CreateDocument(data)
    assert.NotNil(t, doc)
    assert.NoError(t, err)
}

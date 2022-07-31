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

    assert.Equal(t, "3.0.1", doc.Version.Value)
    assert.Equal(t, "Burger Shop", doc.Info.Value.Title.Value)
    assert.NotEmpty(t, doc.Info.Value.Title.Value)
    assert.Equal(t, "https://pb33f.io", doc.Info.Value.TermsOfService.Value)
    assert.Equal(t, "pb33f", doc.Info.Value.Contact.Value.Name.Value)
    assert.Equal(t, "buckaroo@pb33f.io", doc.Info.Value.Contact.Value.Email.Value)
    assert.Equal(t, "https://pb33f.io", doc.Info.Value.Contact.Value.URL.Value)
    assert.Equal(t, "1.2", doc.Info.Value.Version.Value)

}

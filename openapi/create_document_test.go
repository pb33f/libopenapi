package openapi

import (
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/low/3.0"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

var doc *v3.Document

func init() {
	data, _ := ioutil.ReadFile("../test_specs/burgershop.openapi.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	doc, _ = CreateDocument(info)
}

func BenchmarkCreateDocument(b *testing.B) {
	data, _ := ioutil.ReadFile("../test_specs/burgershop.openapi.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	for i := 0; i < b.N; i++ {
		doc, _ = CreateDocument(info)
	}
}

func BenchmarkCreateDocument_Stripe(b *testing.B) {
	data, _ := ioutil.ReadFile("../test_specs/stripe.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	for i := 0; i < b.N; i++ {
		_, err := CreateDocument(info)
		if err != nil {
			panic("this should not error")
		}
	}
}

func BenchmarkCreateDocument_Petstore(b *testing.B) {
	data, _ := ioutil.ReadFile("../test_specs/petstorev3.json")
	info, _ := datamodel.ExtractSpecInfo(data)
	for i := 0; i < b.N; i++ {
		_, err := CreateDocument(info)
		if err != nil {
			panic("this should not error")
		}
	}
}

func TestCreateDocument(t *testing.T) {
	assert.Equal(t, "3.0.1", doc.Version.Value)
	assert.Equal(t, "Burger Shop", doc.Info.Value.Title.Value)
	assert.NotEmpty(t, doc.Info.Value.Title.Value)
}

func TestCreateDocument_Info(t *testing.T) {
	assert.Equal(t, "https://pb33f.io", doc.Info.Value.TermsOfService.Value)
	assert.Equal(t, "pb33f", doc.Info.Value.Contact.Value.Name.Value)
	assert.Equal(t, "buckaroo@pb33f.io", doc.Info.Value.Contact.Value.Email.Value)
	assert.Equal(t, "https://pb33f.io", doc.Info.Value.Contact.Value.URL.Value)
	assert.Equal(t, "pb33f", doc.Info.Value.License.Value.Name.Value)
	assert.Equal(t, "https://pb33f.io/made-up", doc.Info.Value.License.Value.URL.Value)
}

func TestCreateDocument_Servers(t *testing.T) {
	assert.Len(t, doc.Servers.Value, 2)
	server1 := doc.Servers.Value[0].Value
	server2 := doc.Servers.Value[1].Value

	// server 1
	assert.Equal(t, "{scheme}://api.pb33f.io", server1.URL.Value)
	assert.NotEmpty(t, server1.Description.Value)
	assert.Len(t, server1.Variables.Value, 1)
	assert.Len(t, server1.Variables.Value["scheme"].Value.Enum, 2)
	assert.Equal(t, server1.Variables.Value["scheme"].Value.Default.Value, "https")
	assert.NotEmpty(t, server1.Variables.Value["scheme"].Value.Description.Value)

	// server 2
	assert.Equal(t, "https://{domain}.{host}.com", server2.URL.Value)
	assert.NotEmpty(t, server2.Description.Value)
	assert.Len(t, server2.Variables.Value, 2)
	assert.Equal(t, server2.Variables.Value["domain"].Value.Default.Value, "api")
	assert.NotEmpty(t, server2.Variables.Value["domain"].Value.Description.Value)
	assert.NotEmpty(t, server2.Variables.Value["host"].Value.Description.Value)
	assert.Equal(t, server2.Variables.Value["host"].Value.Default.Value, "pb33f.io")
	assert.Equal(t, "1.2", doc.Info.Value.Version.Value)
}

func TestCreateDocument_Tags(t *testing.T) {
	assert.Len(t, doc.Tags.Value, 2)

	// tag1
	assert.Equal(t, "Burgers", doc.Tags.Value[0].Value.Name.Value)
	assert.NotEmpty(t, doc.Tags.Value[0].Value.Description.Value)
	assert.NotNil(t, doc.Tags.Value[0].Value.ExternalDocs.Value)
	assert.Equal(t, "https://pb33f.io", doc.Tags.Value[0].Value.ExternalDocs.Value.URL.Value)
	assert.NotEmpty(t, doc.Tags.Value[0].Value.ExternalDocs.Value.URL.Value)
	assert.Len(t, doc.Tags.Value[0].Value.Extensions, 7)

	for key, extension := range doc.Tags.Value[0].Value.Extensions {
		switch key.Value {
		case "x-internal-ting":
			assert.Equal(t, "somethingSpecial", extension.Value)
		case "x-internal-tong":
			assert.Equal(t, int64(1), extension.Value)
		case "x-internal-tang":
			assert.Equal(t, 1.2, extension.Value)
		case "x-internal-tung":
			assert.Equal(t, true, extension.Value)
		case "x-internal-arr":
			assert.Len(t, extension.Value, 2)
			assert.Equal(t, "one", extension.Value.([]interface{})[0].(string))
		case "x-internal-arrmap":
			assert.Len(t, extension.Value, 2)
			assert.Equal(t, "now", extension.Value.([]interface{})[0].(map[string]interface{})["what"])
		case "x-something-else":
			// crazy times in the upside down. this API should be avoided for the higher up use cases.
			// this is why we will need a higher level API to this model, this looks cool and all, but dude.
			assert.Equal(t, "now?", extension.Value.(map[string]interface{})["ok"].([]interface{})[0].(map[string]interface{})["what"])
		}

	}

	/// tag2
	assert.Equal(t, "Dressing", doc.Tags.Value[1].Value.Name.Value)
	assert.NotEmpty(t, doc.Tags.Value[1].Value.Description.Value)
	assert.NotNil(t, doc.Tags.Value[1].Value.ExternalDocs.Value)
	assert.Equal(t, "https://pb33f.io", doc.Tags.Value[1].Value.ExternalDocs.Value.URL.Value)
	assert.NotEmpty(t, doc.Tags.Value[1].Value.ExternalDocs.Value.URL.Value)
	assert.Len(t, doc.Tags.Value[1].Value.Extensions, 0)

}

func TestCreateDocument_Paths(t *testing.T) {
	doc := doc
	assert.Len(t, doc.Paths.Value.PathItems, 6)
	burgerId := doc.Paths.Value.FindPath("/burgers/{burgerId}")
	assert.NotNil(t, burgerId)
	assert.Len(t, burgerId.Value.Get.Value.Parameters.Value, 2)
	param := burgerId.Value.Get.Value.Parameters.Value[1]
	assert.Equal(t, "burgerHeader", param.Value.Name.Value)
	prop := param.Value.Schema.Value.FindProperty("burgerTheme")
	assert.Equal(t, "something about a theme?", prop.Value.Description.Value)
	assert.Equal(t, "big-mac", param.Value.Example.Value)

	// check content
	pContent := param.Value.FindContent("application/json")
	assert.Equal(t, "somethingNice", pContent.Value.Example.Value)

	encoding := pContent.Value.FindPropertyEncoding("burgerTheme")
	assert.NotNil(t, encoding.Value)
	assert.Len(t, encoding.Value.Headers.Value, 1)

	header := encoding.Value.FindHeader("someHeader")
	assert.NotNil(t, header.Value)
	assert.Equal(t, "this is a header", header.Value.Description.Value)
	assert.Equal(t, "string", header.Value.Schema.Value.Type.Value)

	// check request body on operation
	burgers := doc.Paths.Value.FindPath("/burgers")
	assert.NotNil(t, burgers.Value.Post.Value)

	burgersPost := burgers.Value.Post.Value
	assert.Equal(t, "createBurger", burgersPost.OperationId.Value)
	assert.Equal(t, "Create a new burger", burgersPost.Summary.Value)
	assert.NotEmpty(t, burgersPost.Description.Value)

	requestBody := burgersPost.RequestBody.Value

	assert.NotEmpty(t, requestBody.Description.Value)
	content := requestBody.FindContent("application/json").Value

	assert.NotNil(t, content)
	assert.Len(t, content.Schema.Value.Properties.Value, 4)
	assert.Len(t, content.GetAllExamples(), 2)

	ex := content.FindExample("pbjBurger")
	assert.NotNil(t, ex.Value)
	assert.NotEmpty(t, ex.Value.Summary.Value)
	assert.NotNil(t, ex.Value.Value.Value)

	if n, ok := ex.Value.Value.Value.(map[string]interface{}); ok {
		assert.Len(t, n, 2)
		assert.Equal(t, 3, n["numPatties"])
	} else {
		assert.Fail(t, "should easily be convertable. something changed!")
	}

	cb := content.FindExample("cakeBurger")
	assert.NotNil(t, cb.Value)
	assert.NotEmpty(t, cb.Value.Summary.Value)
	assert.NotNil(t, cb.Value.Value.Value)

	if n, ok := cb.Value.Value.Value.(map[string]interface{}); ok {
		assert.Len(t, n, 2)
		assert.Equal(t, "Chocolate Cake Burger", n["name"])
		assert.Equal(t, 5, n["numPatties"])
	} else {
		assert.Fail(t, "should easily be convertable. something changed!")
	}

	// check responses
	responses := burgersPost.Responses.Value
	assert.NotNil(t, responses)
	assert.Len(t, responses.Codes, 3)

	okCode := responses.FindResponseByCode("200")
	assert.NotNil(t, okCode.Value)
	assert.Equal(t, "A tasty burger for you to eat.", okCode.Value.Description.Value)

	// check headers are populated
	assert.Len(t, okCode.Value.Headers.Value, 1)
	okheader := okCode.Value.FindHeader("UseOil")
	assert.NotNil(t, okheader.Value)
	assert.Equal(t, "this is a header", okheader.Value.Description.Value)

	respContent := okCode.Value.FindContent("application/json").Value
	assert.NotNil(t, respContent)

	assert.NotNil(t, respContent.Schema.Value)
	assert.Len(t, respContent.Schema.Value.Required.Value, 2)

	respExample := respContent.FindExample("quarterPounder")
	assert.NotNil(t, respExample.Value)
	assert.NotNil(t, respExample.Value.Value.Value)

	if n, ok := respExample.Value.Value.Value.(map[string]interface{}); ok {
		assert.Len(t, n, 2)
		assert.Equal(t, "Quarter Pounder with Cheese", n["name"])
		assert.Equal(t, 1, n["numPatties"])
	} else {
		assert.Fail(t, "should easily be convertable. something changed!")
	}

	// check links
	links := okCode.Value.Links
	assert.NotNil(t, links.Value)
	assert.Len(t, links.Value, 2)
	assert.Equal(t, "locateBurger", okCode.Value.FindLink("LocateBurger").Value.OperationId.Value)

	locateBurger := okCode.Value.FindLink("LocateBurger").Value

	burgerIdParam := locateBurger.FindParameter("burgerId")
	assert.NotNil(t, burgerIdParam)
	assert.Equal(t, "$response.body#/id", burgerIdParam.Value)

	// check security requirements
	security := burgersPost.Security.Value
	assert.NotNil(t, security)
	assert.Len(t, security.ValueRequirements, 1)

	oAuthReq := security.FindRequirement("OAuthScheme")
	assert.Len(t, oAuthReq, 2)
	assert.Equal(t, "read:burgers", oAuthReq[0].Value)

	servers := burgersPost.Servers.Value
	assert.NotNil(t, servers)
	assert.Len(t, servers, 1)
	assert.Equal(t, "https://pb33f.io", servers[0].Value.URL.Value)

}

func TestCreateDocument_Components_Schemas(t *testing.T) {

	components := doc.Components.Value
	assert.NotNil(t, components)
	assert.Len(t, components.Schemas.Value, 6)

	burger := components.FindSchema("Burger")
	assert.NotNil(t, burger.Value)
	assert.Equal(t, "The tastiest food on the planet you would love to eat everyday", burger.Value.Description.Value)

	er := components.FindSchema("Error")
	assert.NotNil(t, er.Value)
	assert.Equal(t, "Error defining what went wrong when providing a specification. The message should help indicate the issue clearly.", er.Value.Description.Value)

	fries := components.FindSchema("Fries")
	assert.NotNil(t, fries.Value)

	assert.Len(t, fries.Value.Properties.Value, 3)
	p := fries.Value.FindProperty("favoriteDrink")
	assert.Equal(t, "a frosty cold beverage can be coke or sprite",
		p.Value.Description.Value)

}

func TestCreateDocument_Components_SecuritySchemes(t *testing.T) {
	components := doc.Components.Value
	securitySchemes := components.SecuritySchemes.Value
	assert.Len(t, securitySchemes, 3)

	apiKey := components.FindSecurityScheme("APIKeyScheme").Value
	assert.NotNil(t, apiKey)
	assert.Equal(t, "an apiKey security scheme", apiKey.Description.Value)

	oAuth := components.FindSecurityScheme("OAuthScheme").Value
	assert.NotNil(t, oAuth)
	assert.Equal(t, "an oAuth security scheme", oAuth.Description.Value)
	assert.NotNil(t, oAuth.Flows.Value.Implicit.Value)
	assert.NotNil(t, oAuth.Flows.Value.AuthorizationCode.Value)

	scopes := oAuth.Flows.Value.Implicit.Value.Scopes.Value
	assert.NotNil(t, scopes)

	readScope := oAuth.Flows.Value.Implicit.Value.FindScope("write:burgers")
	assert.NotNil(t, readScope)
	assert.Equal(t, "modify and add new burgers", readScope.Value)

	readScope = oAuth.Flows.Value.AuthorizationCode.Value.FindScope("write:burgers")
	assert.NotNil(t, readScope)
	assert.Equal(t, "modify burgers and stuff", readScope.Value)

}

func TestCreateDocument_Components_Responses(t *testing.T) {
	components := doc.Components.Value
	responses := components.Responses.Value
	assert.Len(t, responses, 1)

	dressingResponse := components.FindResponse("DressingResponse")
	assert.NotNil(t, dressingResponse.Value)
	assert.Equal(t, "all the dressings for a burger.", dressingResponse.Value.Description.Value)
	assert.Len(t, dressingResponse.Value.Content.Value, 1)

}

func TestCreateDocument_Components_Examples(t *testing.T) {
	components := doc.Components.Value
	examples := components.Examples.Value
	assert.Len(t, examples, 1)

	quarterPounder := components.FindExample("QuarterPounder")
	assert.NotNil(t, quarterPounder.Value)
	assert.Equal(t, "A juicy two hander sammich", quarterPounder.Value.Summary.Value)
	assert.NotNil(t, quarterPounder.Value.Value.Value)
}

func TestCreateDocument_Components_RequestBodies(t *testing.T) {
	components := doc.Components.Value
	requestBodies := components.RequestBodies.Value
	assert.Len(t, requestBodies, 1)

	burgerRequest := components.FindRequestBody("BurgerRequest")
	assert.NotNil(t, burgerRequest.Value)
	assert.Equal(t, "Give us the new burger!", burgerRequest.Value.Description.Value)
	assert.Len(t, burgerRequest.Value.Content.Value, 1)
}

func TestCreateDocument_Components_Headers(t *testing.T) {
	components := doc.Components.Value
	headers := components.Headers.Value
	assert.Len(t, headers, 1)

	useOil := components.FindHeader("UseOil")
	assert.NotNil(t, useOil.Value)
	assert.Equal(t, "this is a header", useOil.Value.Description.Value)
	assert.Equal(t, "string", useOil.Value.Schema.Value.Type.Value)
}

func TestCreateDocument_Components_Links(t *testing.T) {
	components := doc.Components.Value
	links := components.Links.Value
	assert.Len(t, links, 2)

	locateBurger := components.FindLink("LocateBurger")
	assert.NotNil(t, locateBurger.Value)
	assert.Equal(t, "Go and get a tasty burger", locateBurger.Value.Description.Value)

	anotherLocateBurger := components.FindLink("AnotherLocateBurger")
	assert.NotNil(t, anotherLocateBurger.Value)
	assert.Equal(t, "Go and get another really tasty burger", anotherLocateBurger.Value.Description.Value)
}

func TestCreateDocument_Doc_Security(t *testing.T) {
	security := doc.Security.Value
	assert.NotNil(t, security)
	assert.Len(t, security.ValueRequirements, 1)

	oAuth := security.FindRequirement("OAuthScheme")
	assert.Len(t, oAuth, 2)
}

func TestCreateDocument_Callbacks(t *testing.T) {
	callbacks := doc.Components.Value.Callbacks.Value
	assert.Len(t, callbacks, 1)

	bCallback := doc.Components.Value.FindCallback("BurgerCallback")
	assert.NotNil(t, bCallback.Value)
	assert.Len(t, callbacks, 1)

	exp := bCallback.Value.FindExpression("{$request.query.queryUrl}")
	assert.NotNil(t, exp.Value)
	assert.NotNil(t, exp.Value.Post.Value)
	assert.Equal(t, "Callback payload", exp.Value.Post.Value.RequestBody.Value.Description.Value)
}

func TestCreateDocument_Component_Discriminator(t *testing.T) {

	components := doc.Components.Value
	dsc := components.FindSchema("Drink").Value.Discriminator.Value
	assert.NotNil(t, dsc)
	assert.Equal(t, "drinkType", dsc.PropertyName.Value)
	assert.Equal(t, "some value", dsc.FindMappingValue("drink").Value)
	assert.Nil(t, dsc.FindMappingValue("don't exist"))
}

func TestCreateDocument_CheckAdditionalProperties_Schema(t *testing.T) {
	components := doc.Components.Value
	d := components.FindSchema("Dressing")
	assert.NotNil(t, d.Value.AdditionalProperties.Value)
	if n, ok := d.Value.AdditionalProperties.Value.(*v3.Schema); ok {
		assert.Equal(t, "something in here.", n.Description.Value)
	} else {
		assert.Fail(t, "should be a schema")
	}
}

func TestCreateDocument_CheckAdditionalProperties_Bool(t *testing.T) {
	components := doc.Components.Value
	d := components.FindSchema("Drink")
	assert.NotNil(t, d.Value.AdditionalProperties.Value)
	assert.True(t, d.Value.AdditionalProperties.Value.(bool))
}

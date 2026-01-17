package v3

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var doc *Document

func initTest() {
	if doc != nil {
		return
	}
	data, _ := os.ReadFile("../../../test_specs/burgershop.openapi.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	var err error
	// deprecated function test.
	doc, err = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	if err != nil {
		panic("broken something")
	}
}

func BenchmarkCreateDocument(b *testing.B) {
	data, _ := os.ReadFile("../../../test_specs/burgershop.openapi.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	for i := 0; i < b.N; i++ {
		doc, _ = CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	}
}

func TestCreateDocument_SelfWithHttpURL(t *testing.T) {
	low.ClearHashCache()
	yml := `openapi: 3.2.0
$self: http://pb33f.io/path/to/spec.yaml
info:
  title: Test API
  version: 1.0.0
paths: {}`

	info, err := datamodel.ExtractSpecInfo([]byte(yml))
	require.NoError(t, err)

	// Test without BaseURL config - should use $self as BaseURL
	config := datamodel.NewDocumentConfiguration()
	doc, err := CreateDocumentFromConfig(info, config)
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "http://pb33f.io/path/to/spec.yaml", doc.Self.Value)
	// maphash uses random seed per process, just verify non-zero
	assert.NotEqual(t, uint64(0), doc.Hash())
}

func TestCreateDocument_SelfWithNonHttpURL(t *testing.T) {
	yml := `openapi: 3.2.0
$self: file:///path/to/spec.yaml
info:
  title: Test API
  version: 1.0.0
paths: {}`

	info, err := datamodel.ExtractSpecInfo([]byte(yml))
	require.NoError(t, err)

	// Test without BaseURL config - should use $self as BaseURL
	config := datamodel.NewDocumentConfiguration()
	doc, err := CreateDocumentFromConfig(info, config)
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "file:///path/to/spec.yaml", doc.Self.Value)

	// Test with BaseURL config and Logger - should log error and use BaseURL
	baseURL, _ := url.Parse("https://api.example.com/v1")
	config2 := datamodel.NewDocumentConfiguration()
	config2.BaseURL = baseURL

	// Capture log output
	logBuffer := &testLogHandler{}
	config2.Logger = slog.New(logBuffer)

	doc2, err := CreateDocumentFromConfig(info, config2)
	require.NoError(t, err)
	assert.NotNil(t, doc2)

	// Verify error was logged
	assert.Contains(t, logBuffer.String(), "BaseURL and $self have been set and conflict")
	assert.Contains(t, logBuffer.String(), "defaulting to BaseURL")
}

// testLogHandler is a simple handler for testing log output
type testLogHandler struct {
	messages []string
}

func (h *testLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *testLogHandler) Handle(ctx context.Context, r slog.Record) error {
	h.messages = append(h.messages, r.Message)
	return nil
}

func (h *testLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *testLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *testLogHandler) String() string {
	if len(h.messages) > 0 {
		return h.messages[0]
	}
	return ""
}

func BenchmarkCreateDocument_Circular(b *testing.B) {
	data, _ := os.ReadFile("../../../test_specs/circular-tests.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	for i := 0; i < b.N; i++ {
		_, err := CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
		if err == nil {
			panic("this should error, it has circular references")
		}
	}
}

func TestCircularReferenceError(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/circular-tests.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	circDoc, err := CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())

	assert.NotNil(t, circDoc)
	assert.Error(t, err)

	assert.Len(t, utils.UnwrapErrors(err), 3)
}

func TestRolodexLocalFileSystem(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()
	cf.BasePath = "../../../test_specs"
	cf.FileFilter = []string{"first.yaml", "second.yaml", "third.yaml"}
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.NoError(t, err)
}

func TestRolodexLocalFileSystem_ProvideNonRolodexFS(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	baseDir := "../../../test_specs"

	cf := datamodel.NewDocumentConfiguration()
	cf.BasePath = baseDir
	cf.FileFilter = []string{"first.yaml", "second.yaml", "third.yaml"}
	cf.LocalFS = os.DirFS(baseDir)
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.Error(t, err)
}

func TestRolodexLocalFileSystem_ProvideRolodexFS(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	baseDir := "../../../test_specs"
	cf := datamodel.NewDocumentConfiguration()
	cf.BasePath = baseDir
	cf.FileFilter = []string{"first.yaml", "second.yaml", "third.yaml"}

	localFS, lErr := index.NewLocalFSWithConfig(&index.LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         os.DirFS(baseDir),
		FileFilters:   cf.FileFilter,
	})
	cf.LocalFS = localFS

	assert.NoError(t, lErr)
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.NoError(t, err)
}

func TestRolodexLocalFileSystem_BadPath(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()
	cf.BasePath = "/NOWHERE"
	cf.FileFilter = []string{"first.yaml", "second.yaml", "third.yaml"}
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.Error(t, err)
}

func TestRolodexRemoteFileSystem(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()
	cf.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	baseUrl := "https://raw.githubusercontent.com/pb33f/libopenapi/main/test_specs"
	u, _ := url.Parse(baseUrl)
	cf.BaseURL = u
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.NoError(t, err)
}

func TestRolodexRemoteFileSystem_BadBase(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()

	baseUrl := "https://no-no-this-will-not-work-it-just-will-not-get-the-job-done-mate.com"
	u, _ := url.Parse(baseUrl)
	cf.BaseURL = u
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.Error(t, err)
}

func TestRolodexRemoteFileSystem_CustomRemote_NoBaseURL(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()
	cf.RemoteFS, _ = index.NewRemoteFSWithConfig(&index.SpecIndexConfig{})
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.Error(t, err)
}

func TestRolodexRemoteFileSystem_CustomHttpHandler(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/first.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	cf := datamodel.NewDocumentConfiguration()
	cf.RemoteURLHandler = http.Get
	baseUrl := "https://no-no-this-will-not-work-it-just-will-not-get-the-job-done-mate.com"
	u, _ := url.Parse(baseUrl)
	cf.BaseURL = u

	pizza := func(url string) (resp *http.Response, err error) {
		return nil, nil
	}
	cf.RemoteURLHandler = pizza
	lDoc, err := CreateDocumentFromConfig(info, cf)
	assert.NotNil(t, lDoc)
	assert.Error(t, err)
}

func TestCircularReference_IgnoreArray(t *testing.T) {
	spec := `openapi: 3.1.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "array"
          items:
            $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	info, _ := datamodel.ExtractSpecInfo([]byte(spec))
	circDoc, err := CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{
		IgnoreArrayCircularReferences: true,
	})
	assert.NotNil(t, circDoc)
	assert.Len(t, utils.UnwrapErrors(err), 0)
}

func TestCircularReference_IgnorePoly(t *testing.T) {
	spec := `openapi: 3.1.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          anyOf:
            - $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	info, _ := datamodel.ExtractSpecInfo([]byte(spec))
	circDoc, err := CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{
		IgnorePolymorphicCircularReferences: true,
	})
	assert.NotNil(t, circDoc)
	assert.Len(t, utils.UnwrapErrors(err), 0)
}

func BenchmarkCreateDocument_Stripe(b *testing.B) {
	data, _ := os.ReadFile("../../../test_specs/stripe.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)

	for i := 0; i < b.N; i++ {
		_, err := CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
		if err != nil {
			panic("this should not error")
		}
	}
}

func BenchmarkCreateDocument_Petstore(b *testing.B) {
	data, _ := os.ReadFile("../../../test_specs/petstorev3.json")
	info, _ := datamodel.ExtractSpecInfo(data)
	for i := 0; i < b.N; i++ {
		_, err := CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
		if err != nil {
			panic("this should not error")
		}
	}
}

func TestCreateDocumentStripe(t *testing.T) {
	data, _ := os.ReadFile("../../../test_specs/stripe.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	d, err := CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	assert.Len(t, utils.UnwrapErrors(err), 1)

	assert.Equal(t, "3.0.0", d.Version.Value)
	assert.Equal(t, "Stripe API", d.Info.Value.Title.Value)
	assert.NotEmpty(t, d.Info.Value.Title.Value)
}

func TestCreateDocument(t *testing.T) {
	initTest()
	assert.Equal(t, "3.1.0", doc.Version.Value)
	assert.Equal(t, "Burger Shop", doc.Info.Value.Title.Value)
	assert.NotEmpty(t, doc.Info.Value.Title.Value)
	assert.Equal(t, "https://pb33f.io/schema", doc.JsonSchemaDialect.Value)
	assert.Equal(t, 1, orderedmap.Len(doc.GetExtensions()))
}

func TestCreateDocumentHash(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()

	data, _ := os.ReadFile("../../../test_specs/all-the-components.yaml")
	info, _ := datamodel.ExtractSpecInfo(data)
	d, _ := CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{
		AllowFileReferences:   false,
		AllowRemoteReferences: false,
		BasePath:              "/here",
	})

	dataB, _ := os.ReadFile("../../../test_specs/all-the-components.yaml")
	infoB, _ := datamodel.ExtractSpecInfo(dataB)
	e, _ := CreateDocumentFromConfig(infoB, &datamodel.DocumentConfiguration{
		AllowFileReferences:   false,
		AllowRemoteReferences: false,
		BasePath:              "/here",
	})

	assert.Equal(t, d.Hash(), e.Hash())
}

func TestCreateDocument_Info(t *testing.T) {
	initTest()
	assert.NotNil(t, doc.GetIndex())
	assert.Equal(t, "https://pb33f.io", doc.Info.Value.TermsOfService.Value)
	assert.Equal(t, "pb33f", doc.Info.Value.Contact.Value.Name.Value)
	assert.Equal(t, "buckaroo@pb33f.io", doc.Info.Value.Contact.Value.Email.Value)
	assert.Equal(t, "https://pb33f.io", doc.Info.Value.Contact.Value.URL.Value)
	assert.Equal(t, "pb33f", doc.Info.Value.License.Value.Name.Value)
	assert.Equal(t, "https://pb33f.io/made-up", doc.Info.Value.License.Value.URL.Value)
}

func TestCreateDocument_WebHooks(t *testing.T) {
	initTest()
	assert.Equal(t, 1, orderedmap.Len(doc.Webhooks.Value))
	for v := range doc.Webhooks.Value.ValuesFromOldest() {
		// a nice deep model should be available for us.
		assert.Equal(t, "Information about a new burger",
			v.Value.Post.Value.RequestBody.Value.Description.Value)
	}
}

func TestCreateDocument_WebHooks_Error(t *testing.T) {
	yml := `openapi: 3.0
webhooks:
      $ref: #bork`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	_, err = CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	assert.Len(t, utils.UnwrapErrors(err), 1)
}

func TestCreateDocument_Servers(t *testing.T) {
	initTest()
	assert.Len(t, doc.Servers.Value, 2)
	server1 := doc.Servers.Value[0].Value
	server2 := doc.Servers.Value[1].Value

	// server 1
	assert.Equal(t, "{scheme}://api.pb33f.io", server1.URL.Value)
	assert.NotEmpty(t, server1.Description.Value)
	assert.Equal(t, 1, orderedmap.Len(server1.Variables.Value))
	assert.Len(t, server1.FindVariable("scheme").Value.Enum, 2)
	assert.Equal(t, server1.FindVariable("scheme").Value.Default.Value, "https")
	assert.NotEmpty(t, server1.FindVariable("scheme").Value.Description.Value)

	// server 2
	assert.Equal(t, "https://{domain}.{host}.com", server2.URL.Value)
	assert.NotEmpty(t, server2.Description.Value)
	assert.Equal(t, 2, orderedmap.Len(server2.Variables.Value))
	assert.Equal(t, "api", server2.FindVariable("domain").Value.Default.Value)
	assert.NotEmpty(t, server2.FindVariable("domain").Value.Description.Value)
	assert.NotEmpty(t, server2.FindVariable("host").Value.Description.Value)
	assert.Equal(t, server2.FindVariable("host").Value.Default.Value, "pb33f.io")
	assert.Equal(t, "1.2", doc.Info.Value.Version.Value)
}

func TestCreateDocument_Tags(t *testing.T) {
	initTest()
	assert.Len(t, doc.Tags.Value, 2)

	// tag1
	assert.Equal(t, "Burgers", doc.Tags.Value[0].Value.Name.Value)
	assert.NotEmpty(t, doc.Tags.Value[0].Value.Description.Value)
	assert.NotNil(t, doc.Tags.Value[0].Value.ExternalDocs.Value)
	assert.Equal(t, "https://pb33f.io", doc.Tags.Value[0].Value.ExternalDocs.Value.URL.Value)
	assert.NotEmpty(t, doc.Tags.Value[0].Value.ExternalDocs.Value.URL.Value)
	assert.Equal(t, 7, orderedmap.Len(doc.Tags.Value[0].Value.Extensions))

	for key, extension := range doc.Tags.Value[0].Value.Extensions.FromOldest() {
		var val any
		_ = extension.Value.Decode(&val)
		switch key.Value {
		case "x-internal-ting":
			assert.Equal(t, "somethingSpecial", val)
		case "x-internal-tong":
			assert.Equal(t, 1, val)
		case "x-internal-tang":
			assert.Equal(t, 1.2, val)
		case "x-internal-tung":
			assert.Equal(t, true, val)
		case "x-internal-arr":
			var a []any
			err := extension.Value.Decode(&a)
			require.NoError(t, err)

			assert.Len(t, a, 2)
			assert.Equal(t, "one", a[0].(string))
		case "x-internal-arrmap":
			var a []any
			err := extension.Value.Decode(&a)
			require.NoError(t, err)

			assert.Len(t, a, 2)
			assert.Equal(t, "now", a[0].(map[string]interface{})["what"])
		case "x-something-else":
			var m map[string]any
			err := extension.Value.Decode(&m)
			require.NoError(t, err)

			// crazy times in the upside down. this API should be avoided for the higher up use cases.
			// this is why we will need a higher level API to this model, this looks cool and all, but dude.
			assert.Equal(t, "now?", m["ok"].([]interface{})[0].(map[string]interface{})["what"])
		}
	}

	/// tag2
	assert.Equal(t, "Dressing", doc.Tags.Value[1].Value.Name.Value)
	assert.NotEmpty(t, doc.Tags.Value[1].Value.Description.Value)
	assert.NotNil(t, doc.Tags.Value[1].Value.ExternalDocs.Value)
	assert.Equal(t, "https://pb33f.io", doc.Tags.Value[1].Value.ExternalDocs.Value.URL.Value)
	assert.NotEmpty(t, doc.Tags.Value[1].Value.ExternalDocs.Value.URL.Value)
	assert.Equal(t, 0, orderedmap.Len(doc.Tags.Value[1].Value.Extensions))
}

func TestCreateDocument_Paths(t *testing.T) {
	initTest()
	assert.Equal(t, 5, orderedmap.Len(doc.Paths.Value.PathItems))
	burgerId := doc.Paths.Value.FindPath("/burgers/{burgerId}")
	assert.NotNil(t, burgerId)
	assert.Len(t, burgerId.Value.Get.Value.Parameters.Value, 2)
	param := burgerId.Value.Get.Value.Parameters.Value[1]
	assert.Equal(t, "burgerHeader", param.Value.Name.Value)
	prop := param.Value.Schema.Value.Schema().FindProperty("burgerTheme").Value
	assert.Equal(t, "something about a theme goes in here?", prop.Schema().Description.Value)

	var paramExample string
	_ = param.GetValue().Example.Value.Decode(&paramExample)
	assert.Equal(t, "big-mac", paramExample)

	// check content
	pContent := param.Value.FindContent("application/json")

	var contentExample string
	_ = pContent.Value.Example.Value.Decode(&contentExample)
	assert.Equal(t, "somethingNice", contentExample)

	encoding := pContent.Value.FindPropertyEncoding("burgerTheme")
	assert.NotNil(t, encoding.Value)
	assert.Equal(t, 1, orderedmap.Len(encoding.Value.Headers.Value))

	header := encoding.Value.FindHeader("someHeader")
	assert.NotNil(t, header.Value)
	assert.Equal(t, "this is a header", header.Value.Description.Value)
	assert.Equal(t, "string", header.Value.Schema.Value.Schema().Type.Value.A)

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
	assert.Equal(t, 4, orderedmap.Len(content.Schema.Value.Schema().Properties.Value))
	assert.Equal(t, 2, orderedmap.Len(content.GetAllExamples()))

	ex := content.FindExample("pbjBurger")
	assert.NotNil(t, ex.Value)
	assert.NotEmpty(t, ex.Value.Summary.Value)
	assert.NotNil(t, ex.Value.Value.Value)

	var pbjBurgerExample map[string]any
	err := ex.Value.Value.Value.Decode(&pbjBurgerExample)
	require.NoError(t, err)

	assert.Len(t, pbjBurgerExample, 2)
	assert.Equal(t, 3, pbjBurgerExample["numPatties"])

	cb := content.FindExample("cakeBurger")
	assert.NotNil(t, cb.Value)
	assert.NotEmpty(t, cb.Value.Summary.Value)
	assert.NotNil(t, cb.Value.Value.Value)

	var cakeBurgerExample map[string]any
	err = cb.Value.Value.Value.Decode(&cakeBurgerExample)
	require.NoError(t, err)

	assert.Len(t, cakeBurgerExample, 2)
	assert.Equal(t, "Chocolate Cake Burger", cakeBurgerExample["name"])
	assert.Equal(t, 5, cakeBurgerExample["numPatties"])

	// check responses
	responses := burgersPost.Responses.Value
	assert.NotNil(t, responses)
	assert.Equal(t, 3, orderedmap.Len(responses.Codes))

	okCode := responses.FindResponseByCode("200")
	assert.NotNil(t, okCode.Value)
	assert.Equal(t, "A tasty burger for you to eat.", okCode.Value.Description.Value)

	// check headers are populated
	assert.Equal(t, 1, orderedmap.Len(okCode.Value.Headers.Value))
	okheader := okCode.Value.FindHeader("UseOil")
	assert.NotNil(t, okheader.Value)
	assert.Equal(t, "this is a header example for UseOil", okheader.Value.Description.Value)

	respContent := okCode.Value.FindContent("application/json").Value
	assert.NotNil(t, respContent)

	assert.NotNil(t, respContent.Schema.Value)
	assert.Len(t, respContent.Schema.Value.Schema().Required.Value, 2)

	respExample := respContent.FindExample("quarterPounder")
	assert.NotNil(t, respExample.Value)
	assert.NotNil(t, respExample.Value.Value.Value)

	var quarterPounderExample map[string]any
	err = respExample.Value.Value.Value.Decode(&quarterPounderExample)
	require.NoError(t, err)

	assert.Len(t, quarterPounderExample, 2)
	assert.Equal(t, "Quarter Pounder with Cheese", quarterPounderExample["name"])
	assert.Equal(t, 1, quarterPounderExample["numPatties"])

	// check links
	links := okCode.Value.Links
	assert.NotNil(t, links.Value)
	assert.Equal(t, 2, orderedmap.Len(links.Value))
	assert.Equal(t, "locateBurger", okCode.Value.FindLink("LocateBurger").Value.OperationId.Value)

	locateBurger := okCode.Value.FindLink("LocateBurger").Value

	burgerIdParam := locateBurger.FindParameter("burgerId")
	assert.NotNil(t, burgerIdParam)
	assert.Equal(t, "$response.body#/id", burgerIdParam.Value)

	// check security requirements
	oAuthReq := burgersPost.FindSecurityRequirement("OAuthScheme")
	assert.Len(t, oAuthReq, 2)
	assert.Equal(t, "read:burgers", oAuthReq[0].Value)

	servers := burgersPost.Servers.Value
	assert.NotNil(t, servers)
	assert.Len(t, servers, 1)
	assert.Equal(t, "https://pb33f.io", servers[0].Value.URL.Value)
}

func TestCreateDocument_Components_Schemas(t *testing.T) {
	initTest()

	components := doc.Components.Value
	assert.NotNil(t, components)
	assert.Equal(t, 6, components.Schemas.Value.Len())

	burger := components.FindSchema("Burger").Value
	assert.NotNil(t, burger)
	assert.Equal(t, "The tastiest food on the planet you would love to eat everyday", burger.Schema().Description.Value)

	er := components.FindSchema("Error")
	assert.NotNil(t, er.Value)
	assert.Equal(t, "Error defining what went wrong when providing a specification. The message should help "+
		"indicate the issue clearly.", er.Value.Schema().Description.Value)

	fries := components.FindSchema("Fries")
	assert.NotNil(t, fries.Value)

	assert.Equal(t, 3, fries.Value.Schema().Properties.Value.Len())
	p := fries.Value.Schema().FindProperty("favoriteDrink")
	assert.Equal(t, "a frosty cold beverage can be coke or sprite",
		p.Value.Schema().Description.Value)
}

func TestCreateDocument_Components_SecuritySchemes(t *testing.T) {
	initTest()
	components := doc.Components.Value
	securitySchemes := components.SecuritySchemes.Value
	assert.Equal(t, 3, securitySchemes.Len())

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
	initTest()
	components := doc.Components.Value
	responses := components.Responses.Value
	assert.Equal(t, 1, responses.Len())

	dressingResponse := components.FindResponse("DressingResponse")
	assert.NotNil(t, dressingResponse.Value)
	assert.Equal(t, "all the dressings for a burger.", dressingResponse.Value.Description.Value)
	assert.Equal(t, 1, dressingResponse.Value.Content.Value.Len())
}

func TestCreateDocument_Components_Examples(t *testing.T) {
	initTest()
	components := doc.Components.Value
	examples := components.Examples.Value
	assert.Equal(t, 1, examples.Len())

	quarterPounder := components.FindExample("QuarterPounder")
	assert.NotNil(t, quarterPounder.Value)
	assert.Equal(t, "A juicy two hander sammich", quarterPounder.Value.Summary.Value)
	assert.NotNil(t, quarterPounder.Value.Value.Value)
}

func TestCreateDocument_Components_RequestBodies(t *testing.T) {
	initTest()
	components := doc.Components.Value
	requestBodies := components.RequestBodies.Value
	assert.Equal(t, 1, requestBodies.Len())

	burgerRequest := components.FindRequestBody("BurgerRequest")
	assert.NotNil(t, burgerRequest.Value)
	assert.Equal(t, "Give us the new burger!", burgerRequest.Value.Description.Value)
	assert.Equal(t, 1, burgerRequest.Value.Content.Value.Len())
}

func TestCreateDocument_Components_Headers(t *testing.T) {
	initTest()
	components := doc.Components.Value
	headers := components.Headers.Value
	assert.Equal(t, 1, headers.Len())

	useOil := components.FindHeader("UseOil")
	assert.NotNil(t, useOil.Value)
	assert.Equal(t, "this is a header example for UseOil", useOil.Value.Description.Value)
	assert.Equal(t, "string", useOil.Value.Schema.Value.Schema().Type.Value.A)
}

func TestCreateDocument_Components_Links(t *testing.T) {
	initTest()
	components := doc.Components.Value
	links := components.Links.Value
	assert.Equal(t, 2, links.Len())

	locateBurger := components.FindLink("LocateBurger")
	assert.NotNil(t, locateBurger.Value)
	assert.Equal(t, "Go and get a tasty burger", locateBurger.Value.Description.Value)

	anotherLocateBurger := components.FindLink("AnotherLocateBurger")
	assert.NotNil(t, anotherLocateBurger.Value)
	assert.Equal(t, "Go and get another really tasty burger", anotherLocateBurger.Value.Description.Value)
}

func TestCreateDocument_Doc_Security(t *testing.T) {
	initTest()
	d := doc
	oAuth := d.FindSecurityRequirement("OAuthScheme")
	assert.Len(t, oAuth, 2)
}

func TestCreateDocument_Callbacks(t *testing.T) {
	initTest()
	callbacks := doc.Components.Value.Callbacks.Value
	assert.Equal(t, 1, callbacks.Len())

	bCallback := doc.Components.Value.FindCallback("BurgerCallback")
	assert.NotNil(t, bCallback.Value)
	assert.Equal(t, 1, callbacks.Len())

	exp := bCallback.Value.FindExpression("{$request.query.queryUrl}")
	assert.NotNil(t, exp.Value)
	assert.NotNil(t, exp.Value.Post.Value)
	assert.Equal(t, "Callback payload", exp.Value.Post.Value.RequestBody.Value.Description.Value)
}

func TestCreateDocument_Component_Discriminator(t *testing.T) {
	initTest()

	components := doc.Components.Value
	dsc := components.FindSchema("Drink").Value.Schema().Discriminator.Value
	assert.NotNil(t, dsc)
	assert.Equal(t, "drinkType", dsc.PropertyName.Value)
	assert.Equal(t, "some value", dsc.FindMappingValue("drink").Value)
	assert.Nil(t, dsc.FindMappingValue("don't exist"))
	assert.NotNil(t, doc.GetExternalDocs())
	assert.Nil(t, doc.FindSecurityRequirement("scooby doo"))
}

func TestCreateDocument_CheckAdditionalProperties_Schema(t *testing.T) {
	initTest()
	components := doc.Components.Value
	d := components.FindSchema("Dressing")
	assert.NotNil(t, d.Value.Schema().AdditionalProperties.Value)

	assert.True(t, d.Value.Schema().AdditionalProperties.Value.IsA(), "should be a schema")
}

func TestCreateDocument_CheckAdditionalProperties_Bool(t *testing.T) {
	initTest()
	components := doc.Components.Value
	d := components.FindSchema("Drink")
	assert.NotNil(t, d.Value.Schema().AdditionalProperties.Value)
	assert.True(t, d.Value.Schema().AdditionalProperties.Value.B)
}

func TestCreateDocument_Components_Error(t *testing.T) {
	yml := `openapi: 3.0
components:
  schemas:
    bork:
      properties:
        bark:
          $ref: #bork`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	assert.NoError(t, err)

	ob := doc.Components.Value.FindSchema("bork").Value
	ob.Schema()
	assert.Error(t, ob.GetBuildError())
}

func TestCreateDocument_Webhooks_Error(t *testing.T) {
	yml := `openapi: 3.0
webhooks:
  aHook:
    $ref: #bork`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	doc, err = CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	assert.Equal(t, "flat map build failed: reference cannot be found: reference at line 4, column 5 is empty, it cannot be resolved",
		err.Error())
}

func TestCreateDocument_Components_Error_Extract(t *testing.T) {
	yml := `openapi: 3.0
components:
  parameters:
    bork:
      $ref: #bork`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	_, err = CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	assert.Equal(t, "schema build failed: reference '[empty]' cannot be found at line 5, col 12", err.Error())
}

func TestCreateDocument_Paths_Errors(t *testing.T) {
	yml := `openapi: 3.0
paths:
  /p:
    $ref: #bork`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	_, err = CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	assert.Equal(t,
		"path item build failed: cannot find reference: '' at line 4, col 10", err.Error())
}

func TestCreateDocument_Tags_Errors(t *testing.T) {
	yml := `openapi: 3.0
tags:
  - $ref: #bork`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	_, err = CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	assert.Equal(t,
		"object extraction failed: reference at line 3, column 5 is empty, it cannot be resolved", err.Error())
}

func TestCreateDocument_Security_Error(t *testing.T) {
	yml := `openapi: 3.0
security:
  $ref: #bork`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	_, err = CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	assert.Equal(t,
		"array build failed: reference cannot be found: reference at line 3, column 3 is empty, it cannot be resolved",
		err.Error())
}

func TestCreateDocument_ExternalDoc_Error(t *testing.T) {
	yml := `openapi: 3.0
externalDocs:
  $ref: #bork`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	_, err = CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	assert.Equal(t, "object extraction failed: reference at line 3, column 3 is empty, it cannot be resolved", err.Error())
}

func TestCreateDocument_YamlAnchor(t *testing.T) {
	// load petstore into bytes
	anchorDocument, _ := os.ReadFile("../../../test_specs/yaml-anchor.yaml")

	// read in specification
	info, _ := datamodel.ExtractSpecInfo(anchorDocument)

	// build low-level document model
	document, err := CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		panic("cannot build document")
	}

	examplePath := document.Paths.GetValue().FindPath("/system/examples/{id}")
	assert.NotNil(t, examplePath)

	// Check tag reference
	getOp := examplePath.GetValue().Get.GetValue()
	assert.NotNil(t, getOp)
	postOp := examplePath.GetValue().Get.GetValue()
	assert.NotNil(t, postOp)
	assert.Equal(t, 1, len(getOp.GetTags().GetValue()))
	assert.Equal(t, 1, len(postOp.GetTags().GetValue()))
	assert.Equal(t, getOp.GetTags().GetValue(), postOp.GetTags().GetValue())

	// Check parameter reference
	getParams := examplePath.Value.Get.Value.Parameters.Value
	assert.NotNil(t, getParams)
	postParams := examplePath.Value.Post.Value.Parameters.Value
	assert.NotNil(t, postParams)
	assert.Equal(t, 1, len(getParams))
	assert.Equal(t, 1, len(postParams))
	assert.Equal(t, getParams[0].ValueNode, postParams[0].ValueNode)

	// check post request body
	responses := examplePath.GetValue().Get.GetValue().GetResponses().Value.(*Responses)
	assert.NotNil(t, responses)
	jsonGet := responses.FindResponseByCode("200").GetValue().FindContent("application/json")
	assert.NotNil(t, jsonGet)

	// Should this work? It doesn't
	// update from quobix 10/14/2023: It does now!
	postJsonType := examplePath.GetValue().Post.GetValue().RequestBody.GetValue().FindContent("application/json")
	assert.NotNil(t, postJsonType)
}

func TestCreateDocument_NotOpenAPI_EnforcedDocCheck(t *testing.T) {
	yml := `notadoc: no`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	var err error
	_, err = CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	assert.Equal(t,
		"no openapi version/tag found, cannot create document", err.Error())
}

func ExampleCreateDocument() {
	// How to create a low-level OpenAPI 3 Document

	// load petstore into bytes
	petstoreBytes, _ := os.ReadFile("../../../test_specs/petstorev3.json")

	// read in specification
	info, _ := datamodel.ExtractSpecInfo(petstoreBytes)

	// build low-level document model
	document, err := CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		panic("cannot build document")
	}

	// print out email address from the info > contact object.
	fmt.Print(document.Info.Value.Contact.Value.Email.Value)
	// Output: apiteam@swagger.io
}

func TestURLWithoutTrailingSlash(t *testing.T) {
	tc := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "url with no path",
			url:  "https://example.com",
			want: "https://example.com",
		},
		{
			name: "nil pointer",
			url:  "",
		},
		{
			name: "URL with path not ending in slash",
			url:  "https://example.com/some/path",
			want: "https://example.com/some/path",
		},
		{
			name: "URL with path ending in slash",
			url:  "https://example.com/some/path/",
			want: "https://example.com/some/path",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse(tt.url)
			if tt.url == "" {
				u = nil
			}

			got := urlWithoutTrailingSlash(u)

			if u == nil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestCreateDocument_WithSelfField(t *testing.T) {
	yml := `openapi: 3.2.0
$self: https://api.example.com/v1/openapi.yaml
info:
  title: Test API
  version: 1.0.0
paths: {}`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	assert.Equal(t, "https://api.example.com/v1/openapi.yaml", info.Self)

	// test document creation extracts $self into low-level model
	doc, err := CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	assert.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "https://api.example.com/v1/openapi.yaml", doc.Self.Value)
	assert.NotNil(t, doc.Self.KeyNode)
	assert.NotNil(t, doc.Self.ValueNode)
}

func TestCreateDocument_WithSelfField_InvalidURL(t *testing.T) {
	yml := `openapi: 3.2.0
$self: not a valid url://
info:
  title: Test API
  version: 1.0.0
paths: {}`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	assert.Equal(t, "not a valid url://", info.Self)

	// create document should still work but log error
	config := datamodel.NewDocumentConfiguration()
	doc, err := CreateDocumentFromConfig(info, config)
	assert.NoError(t, err) // should not fail, just log
	assert.NotNil(t, doc)
	assert.Equal(t, "not a valid url://", doc.Self.Value)
}

func TestCreateDocument_WithSelfField_ConflictWithBaseURL(t *testing.T) {
	yml := `openapi: 3.2.0
$self: https://api.example.com/v1/openapi.yaml
info:
  title: Test API
  version: 1.0.0
paths: {}`

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))

	// configure with a different base URL
	config := datamodel.NewDocumentConfiguration()
	baseURL, _ := url.Parse("https://different.example.com/")
	config.BaseURL = baseURL

	doc, err := CreateDocumentFromConfig(info, config)
	assert.NoError(t, err)
	assert.NotNil(t, doc)

	// programmatic BaseURL should win over $self
	assert.Equal(t, "https://api.example.com/v1/openapi.yaml", doc.Self.Value)
	// but the index should use the configured BaseURL, not $self
	assert.NotNil(t, doc.Index)
}

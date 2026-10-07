// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package gosdk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	modelgen "github.com/pb33f/libopenapi/generator/golang"
	"github.com/pb33f/libopenapi/generator/sdk"
)

// generatePeopleMatch generates a client for a real-world vendor's
// people-enrichment operation, named People.Match.
func generatePeopleMatch(t *testing.T) *sdk.Result {
	t.Helper()
	spec, err := os.ReadFile("testdata/people-match.yaml")
	if err != nil {
		t.Fatal(err)
	}
	document, err := libopenapi.NewDocument(spec)
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sdk.Prepare(&model.Model, sdk.PrepareOptions{
		Operations: []string{"people-enrichment"},
		Names:      map[string]sdk.OperationName{"people-enrichment": {Resource: "People", Method: "Match"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := GenerateContract(contract, Options{PackageName: "people"})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestPeopleMatchGolden(t *testing.T) {
	result := generatePeopleMatch(t)
	for _, file := range result.Files {
		golden := filepath.Join("testdata", "people_match_"+strings.TrimSuffix(file.Path, ".gen.go")+".golden.go")
		if os.Getenv("LIBOPENAPI_GENERATOR_UPDATE_GOLDENS") == "true" {
			if err := os.WriteFile(golden, file.Content, 0o600); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if strings.ReplaceAll(string(want), "\r\n", "\n") != string(file.Content) {
			t.Fatalf("%s does not match %s; regenerate with LIBOPENAPI_GENERATOR_UPDATE_GOLDENS=true and review the diff", file.Path, golden)
		}
	}
	var nullable int
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code != modelgen.DiagnosticNullableKeyword {
			t.Fatalf("unexpected diagnostic %#v", diagnostic)
		}
		nullable++
	}
	if nullable != 3 {
		t.Fatalf("expected the three nullable: true schemas to be reported, got %d", nullable)
	}
}

func TestPeopleMatchClientDecodesDeclaredResponses(t *testing.T) {
	result := generatePeopleMatch(t)
	directory := t.TempDir()
	for _, file := range result.Files {
		if err := os.WriteFile(filepath.Join(directory, file.Path), file.Content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module generated.example/people\n\ngo 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "people_test.go"), []byte(peopleRuntimeTest), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated people client failed: %v\n%s", err, output)
	}
}

const peopleRuntimeTest = `package people

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPeopleMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/people/match" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		switch r.URL.Query().Get("email") {
		case "":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Invalid API key."))
		case "bad@example.com":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(` + "`" + `{"error_details":{"code":"SEARCH.VALIDATION.WEBHOOK_URL_REQUIRED","context":{"parameter":{"value":"reveal_phone_number"}}}}` + "`" + `))
		case "busy@example.com":
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(` + "`" + `{"error_details":{"code":"USAGE.RATE_LIMIT","context":{"window":{"value":"hour"}}}}` + "`" + `))
		default:
			if r.Header.Get("X-Api-Key") != "key" || r.URL.Query().Get("reveal_personal_emails") != "true" {
				t.Errorf("credentials or query not sent: %v %s", r.Header, r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(` + "`" + `{"person":{"name":"Jordan Blake","twitter_url":null,"extrapolated_email_confidence":0.9,"match_confidence":"high","employment_history":[{"_id":"a","id":"b"}],"organization":{"primary_phone":{"number":"+1"}}}}` + "`" + `))
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL+"/api/v1", WithCredential("apiKey", APIKey("key")))
	if err != nil {
		t.Fatal(err)
	}
	email, reveal := "jordan@example.com", true
	response, err := client.People.Match(context.Background(), &PeopleMatchParams{Email: &email, RevealPersonalEmails: &reveal})
	if err != nil {
		t.Fatal(err)
	}
	person := response.Value.Person
	if *person.Name != "Jordan Blake" || string(person.TwitterURL) != "null" || *person.ExtrapolatedEmailConfidence != 0.9 ||
		*person.MatchConfidence != MatchConfidence("high") || *person.EmploymentHistory[0].UnderscoreID != "a" ||
		*person.EmploymentHistory[0].ID != "b" || string(person.Organization.PrimaryPhone) != ` + "`" + `{"number":"+1"}` + "`" + ` {
		t.Fatalf("decoded person = %#v", person)
	}

	var apiError *APIError
	_, err = client.People.Match(context.Background(), nil)
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusUnauthorized || string(apiError.Body) != "Invalid API key." || apiError.Value != nil || apiError.Cause != nil {
		t.Fatalf("text/plain 401 = %#v %v", apiError, err)
	}
	bad := "bad@example.com"
	_, err = client.People.Match(context.Background(), &PeopleMatchParams{Email: &bad})
	if !errors.As(err, &apiError) {
		t.Fatalf("400 = %v", err)
	}
	if value, ok := apiError.Value.(*PeopleMatchBadRequest); !ok || *value.ErrorDetails.Code != "SEARCH.VALIDATION.WEBHOOK_URL_REQUIRED" {
		t.Fatalf("400 value = %#v", apiError.Value)
	}
	busy := "busy@example.com"
	_, err = client.People.Match(context.Background(), &PeopleMatchParams{Email: &busy})
	if !errors.As(err, &apiError) {
		t.Fatalf("429 = %v", err)
	}
	if value, ok := apiError.Value.(*PeopleMatchTooManyRequests); !ok || *value.ErrorDetails.Context["window"].Value != "hour" {
		t.Fatalf("429 value = %#v", apiError.Value)
	}
}
`

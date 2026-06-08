package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"crmservice/internal/config"
)

func TestCollectFilterFieldsIncludesBareFieldKeys(t *testing.T) {
	fields := collectFilterFields(map[string]interface{}{"account_type": "Customer"})
	if len(fields) != 1 {
		t.Fatalf("len(fields) = %d, expected 1", len(fields))
	}
	if fields[0] != "account_type" {
		t.Errorf("fields[0] = %q, expected account_type", fields[0])
	}
}

func TestCollectBodyAttributeFieldsFromFlatInput(t *testing.T) {
	fields, err := collectBodyAttributeFields(&bodyInput{
		body: map[string]interface{}{
			"abc":  "new value",
			"name": "Acme",
		},
	})
	if err != nil {
		t.Fatalf("collectBodyAttributeFields() returned error: %v", err)
	}
	if len(fields) != 2 || fields[0] != "abc" || fields[1] != "name" {
		t.Fatalf("fields = %v, expected [abc name]", fields)
	}
}

func TestValidateModuleAttributesRejectsUnknownField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"}}}`)
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	err := validateModuleAttributes("accounts", []string{"abc"}, url, "token", 0)
	if err == nil {
		t.Fatal("validateModuleAttributes() error = nil, expected unknown field error")
	}
	if !strings.Contains(err.Error(), "abc") {
		t.Fatalf("error = %v, expected abc", err)
	}
}

func TestValidateFilterJSONAgainstModuleRejectsUnknownBareField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"},"account_type":{"type":"string"}}}`)
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	err := validateFilterJSONAgainstModule("accounts", `{"missing_field":"x"}`, url, "token", 0)
	if err == nil {
		t.Fatal("validateFilterJSONAgainstModule() error = nil, expected error")
	}
}

func TestCollectFilterFields(t *testing.T) {
	filter := `{"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}`
	if err := validateFilterJSON(filter); err != nil {
		t.Fatalf("validateFilterJSON() returned error: %v", err)
	}

	var parsed interface{}
	if err := parseFilterJSON(filter, &parsed); err != nil {
		t.Fatalf("parseFilterJSON() returned error: %v", err)
	}

	fields := collectFilterFields(parsed)
	if len(fields) != 2 {
		t.Fatalf("len(fields) = %d, expected 2", len(fields))
	}
}

func TestValidateFilterJSONAgainstModuleRejectsUnknownField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"},"account_type":{"type":"string"}}}`)
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	err := validateFilterJSONAgainstModule("accounts", `{"$eq":["missing_field","x"]}`, url, "token", 0)
	if err == nil {
		t.Fatal("validateFilterJSONAgainstModule() error = nil, expected error")
	}
}

func TestValidateFilterJSONAgainstModuleAcceptsKnownField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"},"account_type":{"type":"string"}}}`)
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	if err := validateFilterJSONAgainstModule("accounts", `{"$eq":["account_type","Customer"]}`, url, "token", 0); err != nil {
		t.Fatalf("validateFilterJSONAgainstModule() returned error: %v", err)
	}
}

func TestValidateFilterJSONAgainstModuleAcceptsKnownBareField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"},"account_type":{"type":"string"}}}`)
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	if err := validateFilterJSONAgainstModule("accounts", `{"account_type":"Customer"}`, url, "token", 0); err != nil {
		t.Fatalf("validateFilterJSONAgainstModule() returned error: %v", err)
	}
}

func TestValidateFilterJSONAgainstModuleValidatesRelatedBareField(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/api/v1/schema/contacts":
			writeTestResponse(t, w, `{"attributes":{"account_id":{"type":"relation","relationModule":"accounts"}}}`)
		case "/api/v1/schema/accounts":
			writeTestResponse(t, w, `{"attributes":{"account_type":{"type":"string"}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	if err := validateFilterJSONAgainstModule("contacts", `{"account.account_type":"Customer"}`, url, "token", 0); err != nil {
		t.Fatalf("validateFilterJSONAgainstModule() returned error: %v", err)
	}
	if requests != 2 {
		t.Errorf("requests = %d, expected 2", requests)
	}
}

func TestValidateFilterJSONAgainstModuleValidatesRelationTypeField(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/api/v1/schema/contacts":
			writeTestResponse(t, w, `{"attributes":{"account_id":{"type":"relation","relationType":"accounts","relationName":"account"}}}`)
		case "/api/v1/schema/accounts":
			writeTestResponse(t, w, `{"attributes":{"account_type":{"type":"string"}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	if err := validateFilterJSONAgainstModule("contacts", `{"$eq":["account.account_type","Customer"]}`, url, "token", 0); err != nil {
		t.Fatalf("validateFilterJSONAgainstModule() returned error: %v", err)
	}
	if requests != 2 {
		t.Errorf("requests = %d, expected 2", requests)
	}
}

func TestValidateFilterJSONAgainstModuleValidatesDeclaredRelation(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/api/v1/schema/accounts":
			writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"}},"relations":{"activities":{"type":"hasMany","class":"activities"}}}`)
		case "/api/v1/schema/activities":
			writeTestResponse(t, w, `{"attributes":{"subject":{"type":"string"}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	if err := validateFilterJSONAgainstModule("accounts", `{"$cts":["activities.subject","review"]}`, url, "token", 0); err != nil {
		t.Fatalf("validateFilterJSONAgainstModule() returned error: %v", err)
	}
	if requests != 2 {
		t.Errorf("requests = %d, expected 2", requests)
	}
}

func TestValidateFilterJSONAgainstModuleValidatesRelatedField(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/api/v1/schema/contacts":
			writeTestResponse(t, w, `{"attributes":{"account_id":{"type":"relation","relationModule":"accounts"}}}`)
		case "/api/v1/schema/accounts":
			writeTestResponse(t, w, `{"attributes":{"account_type":{"type":"string"}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	if err := validateFilterJSONAgainstModule("contacts", `{"$eq":["account.account_type","Customer"]}`, url, "token", 0); err != nil {
		t.Fatalf("validateFilterJSONAgainstModule() returned error: %v", err)
	}
	if requests != 2 {
		t.Errorf("requests = %d, expected 2", requests)
	}
}

func parseFilterJSON(input string, target *interface{}) error {
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	return decoder.Decode(target)
}

package schema

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func writeSchemaTestResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("Write() returned error: %v", err)
	}
}

func TestValidateModuleAttributesRejectsUnknownField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeSchemaTestResponse(t, w, `{"attributes":{"name":{"type":"string"}}}`)
	}))
	defer server.Close()

	validator := &Validator{
		SchemaLoader: func(module string) ([]byte, error) {
			resp, err := http.Get(server.URL + "/api/v1/schema/" + module)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			body := make([]byte, 0, 1024)
			buf := make([]byte, 1024)
			for {
				n, readErr := resp.Body.Read(buf)
				if n > 0 {
					body = append(body, buf[:n]...)
				}
				if readErr != nil {
					break
				}
			}
			return body, nil
		},
	}

	err := validator.ValidateModuleAttributes("accounts", []string{"abc"})
	if err == nil {
		t.Fatal("ValidateModuleAttributes() error = nil, expected unknown field error")
	}
	if !strings.Contains(err.Error(), "abc") {
		t.Fatalf("error = %v, expected abc", err)
	}
}

func TestValidateFilterFieldsRejectsUnknownBareField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeSchemaTestResponse(t, w, `{"attributes":{"name":{"type":"string"},"account_type":{"type":"string"}}}`)
	}))
	defer server.Close()

	validator := newTestValidator(t, server.URL)
	err := validator.ValidateFilterFields("accounts", []string{"missing_field"})
	if err == nil {
		t.Fatal("ValidateFilterFields() error = nil, expected error")
	}
}

func TestValidateFilterFieldsRejectsUnknownField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeSchemaTestResponse(t, w, `{"attributes":{"name":{"type":"string"},"account_type":{"type":"string"}}}`)
	}))
	defer server.Close()

	validator := newTestValidator(t, server.URL)
	err := validator.ValidateFilterFields("accounts", []string{"missing_field"})
	if err == nil {
		t.Fatal("ValidateFilterFields() error = nil, expected error")
	}
}

func TestValidateFilterFieldsAcceptsKnownField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeSchemaTestResponse(t, w, `{"attributes":{"name":{"type":"string"},"account_type":{"type":"string"}}}`)
	}))
	defer server.Close()

	validator := newTestValidator(t, server.URL)
	if err := validator.ValidateFilterFields("accounts", []string{"account_type"}); err != nil {
		t.Fatalf("ValidateFilterFields() returned error: %v", err)
	}
}

func TestValidateFilterFieldsValidatesRelatedField(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/api/v1/schema/contacts":
			writeSchemaTestResponse(t, w, `{"attributes":{"account_id":{"type":"relation","relationModule":"accounts"}}}`)
		case "/api/v1/schema/accounts":
			writeSchemaTestResponse(t, w, `{"attributes":{"account_type":{"type":"string"}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	validator := newTestValidator(t, server.URL)
	if err := validator.ValidateFilterFields("contacts", []string{"account.account_type"}); err != nil {
		t.Fatalf("ValidateFilterFields() returned error: %v", err)
	}
	if requests != 2 {
		t.Errorf("requests = %d, expected 2", requests)
	}
}

func TestValidateFilterFieldsValidatesRelationTypeField(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/api/v1/schema/contacts":
			writeSchemaTestResponse(t, w, `{"attributes":{"account_id":{"type":"relation","relationType":"accounts","relationName":"account"}}}`)
		case "/api/v1/schema/accounts":
			writeSchemaTestResponse(t, w, `{"attributes":{"account_type":{"type":"string"}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	validator := newTestValidator(t, server.URL)
	if err := validator.ValidateFilterFields("contacts", []string{"account.account_type"}); err != nil {
		t.Fatalf("ValidateFilterFields() returned error: %v", err)
	}
	if requests != 2 {
		t.Errorf("requests = %d, expected 2", requests)
	}
}

func TestValidateFilterFieldsValidatesDeclaredRelation(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/api/v1/schema/accounts":
			writeSchemaTestResponse(t, w, `{"attributes":{"name":{"type":"string"}},"relations":{"activities":{"type":"hasMany","class":"activities"}}}`)
		case "/api/v1/schema/activities":
			writeSchemaTestResponse(t, w, `{"attributes":{"subject":{"type":"string"}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	validator := newTestValidator(t, server.URL)
	if err := validator.ValidateFilterFields("accounts", []string{"activities.subject"}); err != nil {
		t.Fatalf("ValidateFilterFields() returned error: %v", err)
	}
	if requests != 2 {
		t.Errorf("requests = %d, expected 2", requests)
	}
}

func newTestValidator(t *testing.T, serverURL string) *Validator {
	t.Helper()
	return &Validator{
		SchemaLoader: func(module string) ([]byte, error) {
			resp, err := http.Get(serverURL + "/api/v1/schema/" + module)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			body := make([]byte, 0, 1024)
			buf := make([]byte, 1024)
			for {
				n, readErr := resp.Body.Read(buf)
				if n > 0 {
					body = append(body, buf[:n]...)
				}
				if readErr != nil {
					break
				}
			}
			return body, nil
		},
	}
}

package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"crmservice/internal/api"
)

func TestWhoamiCmd(t *testing.T) {
	cmd := whoamiCmd()

	if cmd.Use != "whoami" {
		t.Errorf("Use = %q, expected whoami", cmd.Use)
	}
	if cmd.Short != "Show the authenticated CRM user" {
		t.Errorf("Short = %q", cmd.Short)
	}
	for _, name := range []string{"output", "verbose"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("Missing flag: %s", name)
		}
	}
}

func TestFetchCurrentUser(t *testing.T) {
	var authHeader string
	var requestPath string
	var rawQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		requestPath = r.URL.Path
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "123",
				"type": "users",
				"attributes": map[string]interface{}{
					"name":       "Jane Doe",
					"email":      "jane@example.com",
					"is_admin":   true,
					"first_name": "Jane",
					"last_name":  "Doe",
				},
			},
		}); err != nil {
			t.Fatalf("json.Encode() failed: %v", err)
		}
	}))
	defer server.Close()

	user, err := fetchCurrentUser(context.Background(), server.URL+"/api/v1", "token123")
	if err != nil {
		t.Fatalf("fetchCurrentUser() returned error: %v", err)
	}

	if authHeader != "Bearer token123" {
		t.Errorf("Authorization header = %q", authHeader)
	}
	if requestPath != "/api/v1/user" {
		t.Errorf("path = %q", requestPath)
	}
	if rawQuery != "fields[users]=id,name,email,is_admin,first_name,last_name" {
		t.Errorf("query = %q", rawQuery)
	}
	if user["crm_url"] != server.URL+"/api/v1" || user["id"] != "123" || user["email"] != "jane@example.com" || user["is_admin"] != true {
		t.Errorf("unexpected user: %+v", user)
	}
}

func TestIsUnauthorizedError(t *testing.T) {
	if !isUnauthorizedError(&api.Error{Status: http.StatusUnauthorized}) {
		t.Error("401 should be treated as unauthorized")
	}
	if !isUnauthorizedError(&api.Error{Status: http.StatusForbidden}) {
		t.Error("403 should be treated as unauthorized")
	}
	if isUnauthorizedError(&api.Error{Status: http.StatusInternalServerError}) {
		t.Error("500 should not be treated as unauthorized")
	}
}

func TestOutputWhoamiUnauthenticatedJSON(t *testing.T) {
	output := captureStdout(t, func() {
		if err := outputWhoami("json", unauthenticatedWhoami("https://crm.example.com/api/v1", "missing_token")); err != nil {
			t.Fatalf("outputWhoami() returned error: %v", err)
		}
	})

	if !strings.Contains(output, `"authenticated": false`) {
		t.Errorf("expected authenticated=false, got: %s", output)
	}
	if !strings.Contains(output, `"error": "missing_token"`) {
		t.Errorf("expected missing token error, got: %s", output)
	}
	if !strings.Contains(output, `"message": "API token not provided"`) {
		t.Errorf("expected missing token message, got: %s", output)
	}
}

func TestUnauthenticatedWhoamiReasons(t *testing.T) {
	testCases := []struct {
		reason  string
		message string
	}{
		{"missing_url", "CRM URL not provided"},
		{"missing_token", "API token not provided"},
		{"invalid_token", "API token is invalid or unauthorized"},
	}

	for _, tc := range testCases {
		t.Run(tc.reason, func(t *testing.T) {
			data := unauthenticatedWhoami("https://crm.example.com/api/v1", tc.reason)
			if data["error"] != tc.reason {
				t.Errorf("error = %q, expected %q", data["error"], tc.reason)
			}
			if data["message"] != tc.message {
				t.Errorf("message = %q, expected %q", data["message"], tc.message)
			}
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() failed: %v", err)
	}
	return string(data)
}

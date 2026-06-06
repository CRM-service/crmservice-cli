package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://api.example.com/", "my-token")

	if client.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %q, expected %q", client.BaseURL, "https://api.example.com")
	}
	if client.AuthToken != "my-token" {
		t.Errorf("AuthToken = %q, expected %q", client.AuthToken, "my-token")
	}
	if client.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
	if client.HTTPClient.Timeout != 30*time.Second {
		t.Errorf("HTTPClient.Timeout = %v, expected %v", client.HTTPClient.Timeout, 30*time.Second)
	}
	if len(client.DefaultHeaders) != 2 {
		t.Errorf("DefaultHeaders length = %d, expected 2", len(client.DefaultHeaders))
	}
	if client.DefaultHeaders["Content-Type"] != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q", client.DefaultHeaders["Content-Type"])
	}
	if client.DefaultHeaders["Accept"] != "application/vnd.api+json" {
		t.Errorf("Accept = %q", client.DefaultHeaders["Accept"])
	}
}

func TestError_Error(t *testing.T) {
	err := &Error{
		Status:  404,
		Body:    []byte(`{"error":"Not found"}`),
		Message: "Record not found",
	}

	expected := "API error 404: Record not found"
	if err.Error() != expected {
		t.Errorf("Error() = %q, expected %q", err.Error(), expected)
	}
}

func TestNewListOptions(t *testing.T) {
	opts := NewListOptions()

	if opts == nil {
		t.Fatal("NewListOptions returned nil")
		return
	}
	if opts.Page != 0 {
		t.Errorf("Page = %d, expected 0", opts.Page)
	}
	if opts.PageSize != 0 {
		t.Errorf("PageSize = %d, expected 0", opts.PageSize)
	}
	if opts.Offset != 0 {
		t.Errorf("Offset = %d, expected 0", opts.Offset)
	}
}

func TestListOptions_Setters(t *testing.T) {
	opts := &ListOptions{}

	t.Run("SetPageSize", func(t *testing.T) {
		result := opts.SetPageSize(50)
		if result != opts {
			t.Error("SetPageSize should return the same options instance")
		}
		if opts.PageSize != 50 {
			t.Errorf("PageSize = %d, expected 50", opts.PageSize)
		}
	})

	t.Run("SetPage", func(t *testing.T) {
		opts.SetPage(3)
		if opts.Page != 3 {
			t.Errorf("Page = %d, expected 3", opts.Page)
		}
	})

	t.Run("SetOffset", func(t *testing.T) {
		opts.SetOffset(100)
		if opts.Offset != 100 {
			t.Errorf("Offset = %d, expected 100", opts.Offset)
		}
	})

	t.Run("SetFields", func(t *testing.T) {
		fields := []string{"name", "email", "phone"}
		opts.SetFields(fields)
		if len(opts.Fields) != 3 {
			t.Errorf("Fields length = %d, expected 3", len(opts.Fields))
		}
		if opts.Fields[0] != "name" {
			t.Errorf("Fields[0] = %q, expected %q", opts.Fields[0], "name")
		}
	})

	t.Run("SetFilterObj", func(t *testing.T) {
		filter := map[string]interface{}{
			"$and": []interface{}{
				map[string]interface{}{
					"$eq": []interface{}{"name", "John"},
				},
			},
		}
		opts.SetFilterObj(filter)
		if opts.Filter == nil {
			t.Fatal("Filter should not be nil")
		}
		if len(opts.Filter) != 1 {
			t.Errorf("Filter length = %d, expected 1", len(opts.Filter))
		}
	})

	t.Run("AddFilter", func(t *testing.T) {
		opts.AddFilter("status", "active")
		if opts.Filter == nil {
			t.Fatal("Filter should not be nil")
		}
		if opts.Filter["status"] != "active" {
			t.Errorf("Filter[status] = %v, expected %v", opts.Filter["status"], "active")
		}

		opts.AddFilter("type", "customer")
		if opts.Filter["type"] != "customer" {
			t.Errorf("Filter[type] = %v, expected %v", opts.Filter["type"], "customer")
		}
	})

	t.Run("AddInclude", func(t *testing.T) {
		opts.AddInclude("owner")
		if len(opts.Include) != 1 {
			t.Errorf("Include length = %d, expected 1", len(opts.Include))
		}
		if opts.Include[0] != "owner" {
			t.Errorf("Include[0] = %q, expected %q", opts.Include[0], "owner")
		}

		opts.AddInclude("contacts")
		if len(opts.Include) != 2 {
			t.Errorf("Include length = %d, expected 2", len(opts.Include))
		}
		if opts.Include[1] != "contacts" {
			t.Errorf("Include[1] = %q, expected %q", opts.Include[1], "contacts")
		}
	})

	t.Run("SetSort", func(t *testing.T) {
		sort := []string{"name", "-created_at"}
		result := opts.SetSort(sort)
		if result != opts {
			t.Error("SetSort should return the same options instance")
		}
		if len(opts.Sort) != 2 {
			t.Errorf("Sort length = %d, expected 2", len(opts.Sort))
		}
		if opts.Sort[1] != "-created_at" {
			t.Errorf("Sort[1] = %q, expected %q", opts.Sort[1], "-created_at")
		}
	})
}

func TestClient_List(t *testing.T) {
	mockData := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"id":   "1",
				"type": "accounts",
				"attributes": map[string]interface{}{
					"name": "Account 1",
				},
			},
			map[string]interface{}{
				"id":   "2",
				"type": "accounts",
				"attributes": map[string]interface{}{
					"name": "Account 2",
				},
			},
		},
		"meta": map[string]interface{}{
			"total":     2,
			"page":      1,
			"page_size": 20,
		},
		"links": map[string]interface{}{
			"self": "http://example.com/accounts?page[number]=1&page[size]=20",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(mockData); err != nil {
			t.Fatalf("json.Encode() failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")

	t.Run("successful list", func(t *testing.T) {
		resp, err := client.List(context.Background(), "accounts", nil)
		if err != nil {
			t.Fatalf("List() returned error: %v", err)
		}

		if resp.Data == nil {
			t.Fatal("Response Data should not be nil")
		}

		if resp.Meta == nil {
			t.Error("Response Meta should not be nil")
		} else {
			if resp.Meta.Total != 2 {
				t.Errorf("Meta.Total = %d, expected 2", resp.Meta.Total)
			}
			if resp.Meta.Page != 1 {
				t.Errorf("Meta.Page = %d, expected 1", resp.Meta.Page)
			}
		}

		if resp.Links == nil {
			t.Error("Response Links should not be nil")
		}
	})

	t.Run("with options", func(t *testing.T) {
		opts := NewListOptions().
			SetPageSize(10).
			SetPage(2).
			SetOffset(20).
			SetFields([]string{"name", "email"})

		resp, err := client.List(context.Background(), "accounts", opts)
		if err != nil {
			t.Fatalf("List() with options returned error: %v", err)
		}

		if resp.Data == nil {
			t.Fatal("Response Data should not be nil")
		}
	})

	t.Run("with filter", func(t *testing.T) {
		opts := NewListOptions()
		opts.SetFilterObj(map[string]interface{}{"name": "test"})

		resp, err := client.List(context.Background(), "accounts", opts)
		if err != nil {
			t.Fatalf("List() with filter returned error: %v", err)
		}

		if resp.Data == nil {
			t.Fatal("Response Data should not be nil")
		}
	})

	t.Run("with include", func(t *testing.T) {
		opts := NewListOptions()
		opts.AddInclude("owner")

		resp, err := client.List(context.Background(), "accounts", opts)
		if err != nil {
			t.Fatalf("List() with include returned error: %v", err)
		}

		if resp.Data == nil {
			t.Fatal("Response Data should not be nil")
		}
	})

	t.Run("server error", func(t *testing.T) {
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusInternalServerError)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Internal server error",
			}); err != nil {
				t.Fatalf("json.Encode() failed: %v", err)
			}
		}))
		defer errorServer.Close()

		errorClient := NewClient(errorServer.URL, "")
		_, err := errorClient.List(context.Background(), "accounts", nil)
		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})
}

func TestClient_Get(t *testing.T) {
	mockData := map[string]interface{}{
		"data": map[string]interface{}{
			"id":   "123",
			"type": "accounts",
			"attributes": map[string]interface{}{
				"name":  "Test Account",
				"email": "test@example.com",
			},
		},
		"meta": map[string]interface{}{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/accounts/123" {
			t.Errorf("Expected path /accounts/123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(mockData); err != nil {
			t.Fatalf("json.Encode() failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")

	resp, err := client.Get(context.Background(), "accounts", "123", nil)
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}

	if resp.Data == nil {
		t.Fatal("Response Data should not be nil")
	}

	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected Data to be map, got %T", resp.Data)
	}

	if dataMap["id"] != "123" {
		t.Errorf("Data.id = %v, expected 123", dataMap["id"])
	}

	attributes, ok := dataMap["attributes"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected attributes to be map")
	}
	if attributes["name"] != "Test Account" {
		t.Errorf("attributes.name = %v, expected Test Account", attributes["name"])
	}
}

func TestClient_Create(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&receivedBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusCreated)

		reqData, ok := receivedBody["data"].(map[string]interface{})
		if !ok {
			t.Errorf("Expected receivedBody[\"data\"] to be map, got %T", receivedBody["data"])
		}
		attributes, ok := reqData["attributes"].(map[string]interface{})
		if !ok {
			t.Errorf("Expected attributes to be map, got %T", reqData["attributes"])
		}
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":         "new-id",
				"type":       "accounts",
				"attributes": attributes,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("json.Encode() failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")

	data := map[string]interface{}{
		"name":  "New Account",
		"email": "new@example.com",
	}

	resp, err := client.Create(context.Background(), "accounts", data)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if resp.Data == nil {
		t.Fatal("Response Data should not be nil")
	}

	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected Data to be map, got %T", resp.Data)
	}

	if dataMap["type"] != "accounts" {
		t.Errorf("Data.type = %v, expected accounts", dataMap["type"])
	}

	if receivedBody == nil {
		t.Fatal("Request body should not be nil")
	}

	if receivedBody["data"] == nil {
		t.Fatal("Request body data should not be nil")
	}
}

func TestClient_Update(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}
		if r.URL.Path != "/accounts/123" {
			t.Errorf("Expected path /accounts/123, got %s", r.URL.Path)
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&receivedBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)

		reqData, ok := receivedBody["data"].(map[string]interface{})
		if !ok {
			t.Errorf("Expected receivedBody[\"data\"] to be map, got %T", receivedBody["data"])
		}
		attributes, ok := reqData["attributes"].(map[string]interface{})
		if !ok {
			t.Errorf("Expected attributes to be map, got %T", reqData["attributes"])
		}
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":         "123",
				"type":       "accounts",
				"attributes": attributes,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("json.Encode() failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")

	data := map[string]interface{}{
		"name":  "Updated Account",
		"email": "updated@example.com",
	}

	resp, err := client.Update(context.Background(), "accounts", "123", data)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if resp.Data == nil {
		t.Fatal("Response Data should not be nil")
	}

	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected Data to be map, got %T", resp.Data)
	}

	if dataMap["id"] != "123" {
		t.Errorf("Data.id = %v, expected 123", dataMap["id"])
	}

	if receivedBody["data"] == nil {
		t.Fatal("Request body data should not be nil")
	}

	reqData, ok := receivedBody["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected receivedBody[\"data\"] to be map, got %T", receivedBody["data"])
	}
	if reqData["id"] != "123" {
		t.Errorf("Request body data.id = %v, expected 123", reqData["id"])
	}
}

func TestClient_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}
		if r.URL.Path != "/accounts/123" {
			t.Errorf("Expected path /accounts/123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")

	err := client.Delete(context.Background(), "accounts", "123")
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}
}

func TestClient_Do(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "success",
			}); err != nil {
				t.Fatalf("json.Encode() failed: %v", err)
			}
		}))
		defer server.Close()

		client := NewClient(server.URL, "")
		client.Verbose = 0

		var result map[string]interface{}
		err := client.Do(context.Background(), "GET", "/test", nil, &result)
		if err != nil {
			t.Fatalf("Do() returned error: %v", err)
		}

		if result["message"] != "success" {
			t.Errorf("Result message = %v, expected success", result["message"])
		}
	})

	t.Run("POST with body", func(t *testing.T) {
		var receivedBody map[string]interface{}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
				t.Errorf("Failed to decode request body: %v", err)
			}
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"created": true}); err != nil {
				t.Fatalf("json.Encode() failed: %v", err)
			}
		}))
		defer server.Close()

		client := NewClient(server.URL, "")
		body := map[string]interface{}{"name": "test"}
		var result map[string]interface{}
		err := client.Do(context.Background(), "POST", "/test", body, &result)
		if err != nil {
			t.Fatalf("Do() with body returned error: %v", err)
		}

		if receivedBody == nil {
			t.Fatal("Received body should not be nil")
		}
	})

	t.Run("404 error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Not found",
			}); err != nil {
				t.Fatalf("json.Encode() failed: %v", err)
			}
		}))
		defer server.Close()

		client := NewClient(server.URL, "")
		err := client.Do(context.Background(), "GET", "/nonexistent", nil, nil)
		if err == nil {
			t.Error("Expected error for 404 response")
		}

		apiErr, ok := err.(*Error)
		if !ok {
			t.Fatalf("Expected Error, got %T", err)
		}
		if apiErr.Status != 404 {
			t.Errorf("Error.Status = %d, expected 404", apiErr.Status)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Internal server error",
			}); err != nil {
				t.Fatalf("json.Encode() failed: %v", err)
			}
		}))
		defer server.Close()

		client := NewClient(server.URL, "")
		err := client.Do(context.Background(), "GET", "/error", nil, nil)
		if err == nil {
			t.Error("Expected error for 500 response")
		}

		apiErr, ok := err.(*Error)
		if !ok {
			t.Fatalf("Expected Error, got %T", err)
		}
		if apiErr.Status != 500 {
			t.Errorf("Error.Status = %d, expected 500", apiErr.Status)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient(server.URL, "")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := client.Do(ctx, "GET", "/slow", nil, nil)
		if err == nil {
			t.Error("Expected error for cancelled context")
		}
	})

	t.Run("verbose logging", func(t *testing.T) {
		var logOutput string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logOutput += "Request received\n"
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": "test"}); err != nil {
				t.Fatalf("json.Encode() failed: %v", err)
			}
		}))
		defer server.Close()

		client := NewClient(server.URL, "")
		client.Verbose = 2

		var result map[string]interface{}
		err := client.Do(context.Background(), "GET", "/test", nil, &result)
		if err != nil {
			t.Fatalf("Do() with verbose returned error: %v", err)
		}
	})

	t.Run("nil result", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"message": "done"}); err != nil {
				t.Fatalf("json.Encode() failed: %v", err)
			}
		}))
		defer server.Close()

		client := NewClient(server.URL, "")
		err := client.Do(context.Background(), "GET", "/test", nil, nil)
		if err != nil {
			t.Errorf("Do() with nil result returned error: %v", err)
		}
	})
}

func TestClient_AuthToken(t *testing.T) {
	var authHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": "test"}); err != nil {
			t.Fatalf("json.Encode() failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "BearerToken123")

	var result map[string]interface{}
	err := client.Do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("Do() returned error: %v", err)
	}

	expectedAuth := "Bearer BearerToken123"
	if authHeader != expectedAuth {
		t.Errorf("Authorization header = %q, expected %q", authHeader, expectedAuth)
	}
}

func TestClient_DefaultHeaders(t *testing.T) {
	expectedType := "application/vnd.api+json"

	t.Run("POST includes content type and accept", func(t *testing.T) {
		var contentType string
		var accept string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType = r.Header.Get("Content-Type")
			accept = r.Header.Get("Accept")
			w.Header().Set("Content-Type", expectedType)
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": "test"}); err != nil {
				t.Fatalf("json.Encode() failed: %v", err)
			}
		}))
		defer server.Close()

		client := NewClient(server.URL, "")

		var result map[string]interface{}
		err := client.Do(context.Background(), http.MethodPost, "/test", map[string]interface{}{"key": "value"}, &result)
		if err != nil {
			t.Fatalf("Do() returned error: %v", err)
		}

		if contentType != expectedType {
			t.Errorf("Content-Type header = %q, expected %q", contentType, expectedType)
		}
		if accept != expectedType {
			t.Errorf("Accept header = %q, expected %q", accept, expectedType)
		}
	})

	t.Run("GET omits content type and includes accept", func(t *testing.T) {
		var contentType string
		var accept string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType = r.Header.Get("Content-Type")
			accept = r.Header.Get("Accept")
			w.Header().Set("Content-Type", expectedType)
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": "test"}); err != nil {
				t.Fatalf("json.Encode() failed: %v", err)
			}
		}))
		defer server.Close()

		client := NewClient(server.URL, "")

		var result map[string]interface{}
		err := client.Do(context.Background(), http.MethodGet, "/test", nil, &result)
		if err != nil {
			t.Fatalf("Do() returned error: %v", err)
		}

		if contentType != "" {
			t.Errorf("Content-Type header = %q, expected empty", contentType)
		}
		if accept != expectedType {
			t.Errorf("Accept header = %q, expected %q", accept, expectedType)
		}
	})
}

func TestClient_HTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": "test"}); err != nil {
			t.Fatalf("json.Encode() failed: %v", err)
		}
	}))
	defer server.Close()

	customClient := &http.Client{Timeout: 60 * time.Second}
	client := &Client{
		BaseURL:        server.URL,
		HTTPClient:     customClient,
		DefaultHeaders: map[string]string{},
	}

	var result map[string]interface{}
	err := client.Do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("Do() with custom HTTPClient returned error: %v", err)
	}
}

func TestListOptions_Constructors(t *testing.T) {
	t.Run("NewListOptions", func(t *testing.T) {
		opts := NewListOptions()
		if opts == nil {
			t.Fatal("NewListOptions returned nil")
			return
		}
		if opts.Filter != nil {
			t.Error("Filter should be nil initially")
		}
		if opts.Include != nil {
			t.Error("Include should be nil initially")
		}
	})

	t.Run("SetFilterObj clears previous filter", func(t *testing.T) {
		opts := NewListOptions()
		opts.SetFilterObj(map[string]interface{}{"field1": "value1"})
		opts.SetFilterObj(map[string]interface{}{"field2": "value2"})

		if opts.Filter == nil {
			t.Fatal("Filter should not be nil")
		}
		if _, ok := opts.Filter["field1"]; ok {
			t.Error("Old filter key field1 should be removed")
		}
		if _, ok := opts.Filter["field2"]; !ok {
			t.Error("New filter key field2 should exist")
		}
	})

	t.Run("AddFilter accumulates filters", func(t *testing.T) {
		opts := NewListOptions()
		opts.AddFilter("field1", "value1")
		opts.AddFilter("field2", "value2")

		if len(opts.Filter) != 2 {
			t.Errorf("Filter should have 2 keys, got %d", len(opts.Filter))
		}
	})

	t.Run("AddInclude accumulates includes", func(t *testing.T) {
		opts := NewListOptions()
		opts.AddInclude("include1")
		opts.AddInclude("include2")

		if len(opts.Include) != 2 {
			t.Errorf("Include should have 2 items, got %d", len(opts.Include))
		}
	})
}

func TestClient_ListWithOptions(t *testing.T) {
	var receivedQuery string
	var receivedSort string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		receivedSort = r.URL.Query().Get("sort")
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"data":  []interface{}{},
			"meta":  map[string]interface{}{},
			"links": map[string]interface{}{},
		}); err != nil {
			t.Fatalf("json.Encode() failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	opts := NewListOptions()
	opts.SetPageSize(10)
	opts.SetPage(2)
	opts.SetOffset(20)
	opts.SetFields([]string{"name", "email"})
	opts.SetFilterObj(map[string]interface{}{"status": "active"})
	opts.AddInclude("owner")
	opts.SetSort([]string{"name", "-created_at"})

	_, err := client.List(context.Background(), "accounts", opts)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if !strings.Contains(receivedQuery, "page%5Bnumber%5D=2") && !strings.Contains(receivedQuery, "page[number]=2") {
		t.Errorf("Query should contain page[number]=2, got: %s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "page%5Bsize%5D=10") && !strings.Contains(receivedQuery, "page[size]=10") {
		t.Errorf("Query should contain page[size]=10, got: %s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "offset=20") {
		t.Errorf("Query should contain offset=20, got: %s", receivedQuery)
	}
	if receivedSort != "name,-created_at" {
		t.Errorf("Query sort = %q, expected %q", receivedSort, "name,-created_at")
	}
}

func TestResponseUnmarshalsIncluded(t *testing.T) {
	body := []byte(`{
		"data": [],
		"included": [{"id":"u1","type":"users","attributes":{"name":"Owner"}}]
	}`)

	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v", err)
	}
	included, ok := resp.Included.([]interface{})
	if !ok {
		t.Fatalf("Included = %T, expected []interface{}", resp.Included)
	}
	if len(included) != 1 {
		t.Fatalf("len(Included) = %d, expected 1", len(included))
	}
}

func TestClientVerboseLogsToStderr(t *testing.T) {
	client := NewClient("https://api.example.com", "")
	client.Verbose = 1

	stdout := captureAPIStdout(t, func() {
		stderr := captureAPIStderr(t, func() {
			client.logRequest(http.MethodGet, "/accounts")
			client.logResponse(http.StatusOK, []byte(`{"data":[]}`))
		})
		if !strings.Contains(stderr, "[REQUEST]") || !strings.Contains(stderr, "[RESPONSE]") {
			t.Fatalf("stderr = %q, expected request and response logs", stderr)
		}
	})
	if stdout != "" {
		t.Fatalf("stdout = %q, expected empty", stdout)
	}
}

func captureAPIStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	os.Stdout = w
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
	os.Stdout = orig
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() failed: %v", err)
	}
	return string(data)
}

func captureAPIStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	os.Stderr = w
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
	os.Stderr = orig
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() failed: %v", err)
	}
	return string(data)
}

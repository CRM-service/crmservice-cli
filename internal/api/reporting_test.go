package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_Count(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/reporting" {
			t.Errorf("Expected path /reporting, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, expected application/json", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, expected application/json", r.Header.Get("Accept"))
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("json.Decode() returned error: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"meta": map[string]interface{}{
				"results": 1,
				"fields":  []string{"count_value"},
			},
			"data": []map[string]interface{}{
				{"count_value": 42},
			},
		}); err != nil {
			t.Fatalf("json.Encode() failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	count, err := client.Count(context.Background(), "accounts", &CountOptions{
		Filter: map[string]interface{}{
			"$eq": []interface{}{"account_type", "Customer"},
		},
		Include: []string{"owner"},
	})
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 42 {
		t.Errorf("Count() = %d, expected 42", count)
	}

	if receivedBody["from"] != "accounts" {
		t.Errorf("from = %v, expected accounts", receivedBody["from"])
	}
	if receivedBody["select"] != reportingCountSelect {
		t.Errorf("select = %v, expected %q", receivedBody["select"], reportingCountSelect)
	}
	if receivedBody["format"] != "json" {
		t.Errorf("format = %v, expected json", receivedBody["format"])
	}
	if receivedBody["include"] != "owner" {
		t.Errorf("include = %v, expected owner", receivedBody["include"])
	}

	options, ok := receivedBody["options"].(map[string]interface{})
	if !ok {
		t.Fatalf("options = %T, expected map", receivedBody["options"])
	}
	if options["kpi"] != true {
		t.Errorf("options[kpi] = %v, expected true", options["kpi"])
	}

	where, ok := receivedBody["where"].(map[string]interface{})
	if !ok {
		t.Fatalf("where = %T, expected map", receivedBody["where"])
	}
	if _, ok := where["$eq"]; !ok {
		t.Errorf("where = %v, expected $eq filter", where)
	}
}

func TestClient_CountWithoutFilterUsesEmptyArrayWhere(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("json.Decode() returned error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"meta": map[string]interface{}{"results": 1, "fields": []string{"count_value"}},
			"data": []map[string]interface{}{{"count_value": 7}},
		}); err != nil {
			t.Fatalf("json.Encode() failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	count, err := client.Count(context.Background(), "accounts", nil)
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 7 {
		t.Errorf("Count() = %d, expected 7", count)
	}

	where, ok := receivedBody["where"].([]interface{})
	if !ok {
		t.Fatalf("where = %T, expected []interface{}", receivedBody["where"])
	}
	if len(where) != 0 {
		t.Errorf("where length = %d, expected 0", len(where))
	}
}

func TestParseReportingCountUsesCountValueField(t *testing.T) {
	count, err := parseReportingCount(&ReportingResponse{
		Meta: ReportingMeta{Fields: []string{"count_value"}},
		Data: []map[string]interface{}{
			{"count_value": float64(15)},
		},
	})
	if err != nil {
		t.Fatalf("parseReportingCount() returned error: %v", err)
	}
	if count != 15 {
		t.Errorf("count = %d, expected 15", count)
	}
}

func TestParseReportingCountEmptyDataWithResults(t *testing.T) {
	_, err := parseReportingCount(&ReportingResponse{
		Meta: ReportingMeta{Results: 1, Fields: []string{"count_value"}},
		Data: []map[string]interface{}{},
	})
	if err == nil {
		t.Fatal("parseReportingCount() error = nil, expected missing data rows error")
	}
	if !strings.Contains(err.Error(), "missing data rows") {
		t.Fatalf("error = %v, expected missing data rows", err)
	}
}

func TestParseReportingCountEmptyDataWithoutResults(t *testing.T) {
	count, err := parseReportingCount(&ReportingResponse{
		Meta: ReportingMeta{Results: 0, Fields: []string{"count_value"}},
		Data: []map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("parseReportingCount() returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, expected 0", count)
	}
}

func TestParseReportingCountMissingCountValue(t *testing.T) {
	_, err := parseReportingCount(&ReportingResponse{
		Data: []map[string]interface{}{
			{"other": 15},
		},
	})
	if err == nil {
		t.Fatal("parseReportingCount() error = nil, expected missing count_value error")
	}
	if !strings.Contains(err.Error(), "count_value") {
		t.Fatalf("error = %v, expected count_value mention", err)
	}
}

func TestClient_CountServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"message":"Missing 'from'"}`)); err != nil {
			t.Fatalf("Write() failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	_, err := client.Count(context.Background(), "accounts", nil)
	if err == nil {
		t.Fatal("Count() error = nil, expected API error")
	}
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("err = %T, expected *Error", err)
	}
	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("Status = %d, expected 400", apiErr.Status)
	}
	if !strings.Contains(apiErr.Message, "Missing 'from'") {
		t.Errorf("Message = %q, expected Missing 'from'", apiErr.Message)
	}
}

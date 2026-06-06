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
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

func TestListSearchAllFlagsPresent(t *testing.T) {
	for _, cmd := range []*cobra.Command{listCmd(), searchCmd()} {
		for _, name := range []string{"all", "max-results"} {
			if cmd.Flags().Lookup(name) == nil {
				t.Errorf("%s missing flag %s", cmd.Name(), name)
			}
		}
	}
}

func setListFlags(t *testing.T, cmd *cobra.Command, flags map[string]string) {
	t.Helper()
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("Set(%s) error: %v", name, err)
		}
	}
}

func TestValidateListAllFlags(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]string
		wantErr string
	}{
		{name: "max-results without all", flags: map[string]string{"max-results": "100"}, wantErr: "--max-results requires --all"},
		{name: "all without max-results", flags: map[string]string{"all": "true"}, wantErr: "--all requires --max-results"},
		{name: "all with page", flags: map[string]string{"all": "true", "max-results": "100", "page": "2"}, wantErr: "--page cannot be used with --all"},
		{name: "all with offset", flags: map[string]string{"all": "true", "max-results": "100", "offset": "10"}, wantErr: "--offset cannot be used with --all"},
		{name: "valid all unlimited", flags: map[string]string{"all": "true", "max-results": "0"}},
		{name: "valid all capped", flags: map[string]string{"all": "true", "max-results": "500"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := listCmd()
			setListFlags(t, cmd, tt.flags)

			_, _, err := validateListAllFlags(cmd)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateListAllFlags() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateListAllFlags() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestPageSizeForListAllDefault(t *testing.T) {
	cmd := listCmd()
	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatalf("Set(all) error: %v", err)
	}

	pageSize, err := pageSizeForList(cmd, true)
	if err != nil {
		t.Fatalf("pageSizeForList() error: %v", err)
	}
	if pageSize != listAllDefaultPageSize {
		t.Errorf("pageSize = %d, want %d", pageSize, listAllDefaultPageSize)
	}
}

func TestFetchAllListPages(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		page := r.URL.Query().Get("page[number]")
		size := r.URL.Query().Get("page[size]")
		if size != "2" {
			t.Errorf("page[size] = %q, want 2", size)
		}

		var data []map[string]interface{}
		switch page {
		case "1":
			data = []map[string]interface{}{
				{"id": "1", "type": "accounts", "attributes": map[string]interface{}{"name": "A"}},
				{"id": "2", "type": "accounts", "attributes": map[string]interface{}{"name": "B"}},
			}
		case "2":
			data = []map[string]interface{}{
				{"id": "3", "type": "accounts", "attributes": map[string]interface{}{"name": "C"}},
			}
		default:
			data = []map[string]interface{}{}
		}

		if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": data}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "token")
	result, err := fetchAllListPages(context.Background(), client, "accounts", api.NewListOptions(), 2, 0, 0, false)
	if err != nil {
		t.Fatalf("fetchAllListPages() error: %v", err)
	}
	if len(result.data) != 3 {
		t.Errorf("len(data) = %d, want 3", len(result.data))
	}
	if result.truncated {
		t.Error("truncated = true, want false")
	}
	if requests != 2 {
		t.Errorf("requests = %d, want 2", requests)
	}
}

func TestFetchAllListPagesTruncates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []map[string]interface{}{
			{"id": "1", "type": "accounts", "attributes": map[string]interface{}{"name": "A"}},
			{"id": "2", "type": "accounts", "attributes": map[string]interface{}{"name": "B"}},
		}
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": data}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "token")
	result, err := fetchAllListPages(context.Background(), client, "accounts", api.NewListOptions(), 2, 3, 0, false)
	if err != nil {
		t.Fatalf("fetchAllListPages() error: %v", err)
	}
	if len(result.data) != 3 {
		t.Errorf("len(data) = %d, want 3", len(result.data))
	}
	if !result.truncated {
		t.Error("truncated = false, want true")
	}
}

func TestRunListAllOutputsTruncationStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []map[string]interface{}{
			{"id": "1", "type": "accounts", "attributes": map[string]interface{}{"name": "A"}},
			{"id": "2", "type": "accounts", "attributes": map[string]interface{}{"name": "B"}},
		}
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": data}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	cmd := listCmd()
	cmd.SetContext(context.Background())

	client := api.NewClient(server.URL+"/api/v1", "token")
	stdout := captureStdout(t, func() {
		stderr := captureStderr(t, func() {
			if err := runListAll(cmd, client, "accounts", api.NewListOptions(), 100, 1, 0, false, "jsonl", nil); err != nil {
				t.Fatalf("runListAll() error: %v", err)
			}
		})
		if !strings.Contains(stderr, `"status":"truncated"`) && !strings.Contains(stderr, `"status": "truncated"`) {
			t.Fatalf("stderr = %q, expected truncated status", stderr)
		}
	})

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("stdout lines = %d, want 1 record line: %q", len(lines), stdout)
	}
}

func TestRunListAllStreamsJSONL(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		page := r.URL.Query().Get("page[number]")
		var data []map[string]interface{}
		switch page {
		case "1":
			data = []map[string]interface{}{
				{"id": "1", "type": "accounts", "attributes": map[string]interface{}{"name": "A"}},
				{"id": "2", "type": "accounts", "attributes": map[string]interface{}{"name": "B"}},
			}
		case "2":
			data = []map[string]interface{}{
				{"id": "3", "type": "accounts", "attributes": map[string]interface{}{"name": "C"}},
			}
		default:
			data = []map[string]interface{}{}
		}
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": data}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	cmd := listCmd()
	cmd.SetContext(context.Background())
	client := api.NewClient(server.URL+"/api/v1", "token")

	stdout := captureStdout(t, func() {
		if err := runListAll(cmd, client, "accounts", api.NewListOptions(), 2, 0, 0, false, "jsonl", nil); err != nil {
			t.Fatalf("runListAll() error: %v", err)
		}
	})

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("stdout lines = %d, want 3: %q", len(lines), stdout)
	}
	if requests != 2 {
		t.Errorf("requests = %d, want 2", requests)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

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

func TestTruncationStatusFormats(t *testing.T) {
	formats := []struct {
		format  string
		contain string
	}{
		{format: "json", contain: `"status"`},
		{format: "jsonl", contain: `"status"`},
		{format: "yaml", contain: "status:"},
		{format: "csv", contain: "truncated"},
		{format: "table", contain: "Truncated:"},
	}

	for _, tt := range formats {
		t.Run(tt.format, func(t *testing.T) {
			stderr := captureStderr(t, func() {
				if err := output.TruncationStatus(tt.format, 100, 100); err != nil {
					t.Fatalf("TruncationStatus() error: %v", err)
				}
			})
			if !strings.Contains(stderr, tt.contain) {
				t.Fatalf("stderr = %q, want substring %q", stderr, tt.contain)
			}
		})
	}
}

func TestDedupeIncludedResources(t *testing.T) {
	items := []interface{}{
		map[string]interface{}{"id": "1", "type": "users", "attributes": map[string]interface{}{"name": "Owner"}},
		map[string]interface{}{"id": "1", "type": "users", "attributes": map[string]interface{}{"name": "Owner duplicate"}},
		map[string]interface{}{"id": "2", "type": "users", "attributes": map[string]interface{}{"name": "Creator"}},
		map[string]interface{}{"id": "1", "type": "teams", "attributes": map[string]interface{}{"name": "Team"}},
	}

	deduped := dedupeIncludedResources(items)
	if len(deduped) != 3 {
		t.Fatalf("len(deduped) = %d, want 3", len(deduped))
	}
	first, ok := deduped[0].(map[string]interface{})
	if !ok {
		t.Fatalf("deduped[0] = %T, want map", deduped[0])
	}
	attrs, ok := first["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("first[attributes] = %T, want map", first["attributes"])
	}
	if attrs["name"] != "Owner" {
		t.Errorf("first duplicate should be retained, got attributes: %v", attrs)
	}
}

func TestFetchAllListPagesExactMaxChecksForMoreRecords(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var data []map[string]interface{}
		if r.URL.Query().Get("offset") == "2" {
			data = []map[string]interface{}{{"id": "3", "type": "accounts", "attributes": map[string]interface{}{"name": "C"}}}
		} else {
			data = []map[string]interface{}{
				{"id": "1", "type": "accounts", "attributes": map[string]interface{}{"name": "A"}},
				{"id": "2", "type": "accounts", "attributes": map[string]interface{}{"name": "B"}},
			}
		}
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": data}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "token")
	result, err := fetchAllListPages(context.Background(), client, "accounts", api.NewListOptions(), 2, 2, 0, false)
	if err != nil {
		t.Fatalf("fetchAllListPages() error: %v", err)
	}
	if !result.truncated {
		t.Error("truncated = false, want true")
	}
	if requests != 2 {
		t.Errorf("requests = %d, want 2", requests)
	}
}

func TestFetchAllListPagesExactMaxNoTruncationWhenNoMoreRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []map[string]interface{}{}
		if r.URL.Query().Get("offset") == "" {
			data = []map[string]interface{}{
				{"id": "1", "type": "accounts", "attributes": map[string]interface{}{"name": "A"}},
				{"id": "2", "type": "accounts", "attributes": map[string]interface{}{"name": "B"}},
			}
		}
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": data}); err != nil {
			t.Fatalf("Encode() error: %v", err)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "token")
	result, err := fetchAllListPages(context.Background(), client, "accounts", api.NewListOptions(), 2, 2, 0, false)
	if err != nil {
		t.Fatalf("fetchAllListPages() error: %v", err)
	}
	if result.truncated {
		t.Error("truncated = true, want false")
	}
}

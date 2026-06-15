package api

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClient_UploadFile(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "upload-test.txt")
	if err := os.WriteFile(tempFile, []byte("hello upload"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/accounts/123/files" {
			t.Errorf("Path = %s, want /accounts/123/files", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Content-Type = %q, want multipart/form-data", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept") != "application/vnd.api+json" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}

		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("MultipartReader() error: %v", err)
		}

		var gotFileName string
		var gotFileContent string
		attributes := map[string]string{}

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextPart() error: %v", err)
			}

			switch part.FormName() {
			case "file":
				gotFileName = part.FileName()
				content, err := io.ReadAll(part)
				if err != nil {
					t.Fatalf("ReadAll(file) error: %v", err)
				}
				gotFileContent = string(content)
			default:
				if strings.HasPrefix(part.FormName(), "attributes[") {
					key := strings.TrimSuffix(strings.TrimPrefix(part.FormName(), "attributes["), "]")
					value, err := io.ReadAll(part)
					if err != nil {
						t.Fatalf("ReadAll(attribute) error: %v", err)
					}
					attributes[key] = string(value)
				}
			}
		}

		if gotFileName != "upload-test.txt" {
			t.Errorf("file name = %q, want upload-test.txt", gotFileName)
		}
		if gotFileContent != "hello upload" {
			t.Errorf("file content = %q, want hello upload", gotFileContent)
		}
		if attributes["file_usage_type"] != "Entity Attachment" {
			t.Errorf("file_usage_type = %q, want Entity Attachment", attributes["file_usage_type"])
		}
		if attributes["file_access_type"] != "Internal" {
			t.Errorf("file_access_type = %q, want Internal", attributes["file_access_type"])
		}

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "999",
				"type": "files",
				"attributes": map[string]interface{}{
					"file_name": "upload-test.txt",
				},
			},
		}); err != nil {
			t.Fatalf("json.Encode() error: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.UploadFile(context.Background(), "/accounts/123/files", FileUploadRequest{
		FilePath: tempFile,
		Attributes: map[string]interface{}{
			"file_usage_type":  "Entity Attachment",
			"file_access_type": "Internal",
		},
	})
	if err != nil {
		t.Fatalf("UploadFile() error: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Data = %T, want map", resp.Data)
	}
	if data["id"] != "999" {
		t.Errorf("data.id = %v, want 999", data["id"])
	}
}

func TestBuildMultipartUploadBodyRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	_, _, err := buildMultipartUploadBody(FileUploadRequest{FilePath: dir})
	if err == nil {
		t.Fatal("buildMultipartUploadBody() error = nil, expected directory error")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Fatalf("error = %v, want directory message", err)
	}
}

func TestFormatUploadAttributeValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{name: "string", value: "Entity Attachment", want: "Entity Attachment"},
		{name: "bool true", value: true, want: "true"},
		{name: "bool false", value: false, want: "false"},
		{name: "int", value: float64(42), want: "42"},
		{name: "nil", value: nil, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatUploadAttributeValue(tc.value); got != tc.want {
				t.Errorf("formatUploadAttributeValue() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildMultipartUploadBodyProducesValidMultipart(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "sample.pdf")
	if err := os.WriteFile(tempFile, []byte("%PDF-1.4"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	body, contentType, err := buildMultipartUploadBody(FileUploadRequest{
		FilePath: tempFile,
		Attributes: map[string]interface{}{
			"file_usage_type": "Entity Attachment",
		},
	})
	if err != nil {
		t.Fatalf("buildMultipartUploadBody() error: %v", err)
	}
	if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
		t.Fatalf("contentType = %q, want multipart/form-data", contentType)
	}

	reader := multipart.NewReader(body, strings.TrimPrefix(contentType, "multipart/form-data; boundary="))
	foundFile := false
	foundAttribute := false
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("NextPart() error: %v", err)
		}
		switch part.FormName() {
		case "file":
			foundFile = true
		case "attributes[file_usage_type]":
			foundAttribute = true
		}
	}
	if !foundFile {
		t.Error("multipart body missing file part")
	}
	if !foundAttribute {
		t.Error("multipart body missing attributes[file_usage_type] part")
	}
}
package output

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"crmservice/internal/api"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}
	os.Stderr = w
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	os.Stderr = origStderr
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}
	return string(data)
}

func TestStderrErrorJSONFormat(t *testing.T) {
	t.Parallel()

	stderr := captureStderr(t, func() {
		if err := StderrError("json", errors.New("test failure")); err == nil {
			t.Fatal("StderrError() error = nil, expected error")
		}
	})

	decoder := json.NewDecoder(strings.NewReader(stderr))
	var payload map[string]interface{}
	if err := decoder.Decode(&payload); err != nil {
		t.Fatalf("json.Decode(stderr) error: %v\nstderr: %s", err, stderr)
	}
	if payload["error"] != true {
		t.Errorf("error = %v, want true", payload["error"])
	}
	if payload["message"] != "test failure" {
		t.Errorf("message = %v", payload["message"])
	}
}

func TestStderrErrorAPIErrorJSONFormat(t *testing.T) {
	t.Parallel()

	apiErr := &api.Error{Status: 404, Message: "Not found"}
	stderr := captureStderr(t, func() {
		if err := StderrError("json", apiErr); err == nil {
			t.Fatal("StderrError() error = nil, expected error")
		}
	})

	decoder := json.NewDecoder(strings.NewReader(stderr))
	var payload map[string]interface{}
	if err := decoder.Decode(&payload); err != nil {
		t.Fatalf("json.Decode(stderr) error: %v\nstderr: %s", err, stderr)
	}
	if payload["status"] != float64(404) {
		t.Errorf("status = %v", payload["status"])
	}
	if payload["message"] != "Not found" {
		t.Errorf("message = %v", payload["message"])
	}
}

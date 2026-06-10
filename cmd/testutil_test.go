package cmd

import (
	"net/http"
	"testing"
)

func writeTestResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()

	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("Write() returned error: %v", err)
	}
}
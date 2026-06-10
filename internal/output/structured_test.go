package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteStructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]interface{}{"ok": true, "version": "1.0"}
	if err := WriteStructured("json", data, StructuredOptions{Destination: &buf}); err != nil {
		t.Fatalf("WriteStructured() returned error: %v", err)
	}
	if !strings.Contains(buf.String(), `"ok": true`) {
		t.Fatalf("WriteStructured() output = %q", buf.String())
	}
}

func TestWriteStructuredTableOmitEmpty(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]interface{}{"valid": true, "module": "", "message": "ok"}
	if err := WriteStructured("table", data, StructuredOptions{
		Destination: &buf,
		Columns:     []string{"valid", "schema_checked", "module", "message"},
		OmitEmpty:   true,
	}); err != nil {
		t.Fatalf("WriteStructured() returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "valid") || !strings.Contains(out, "message") {
		t.Fatalf("WriteStructured() output = %q", out)
	}
	if strings.Contains(out, "module") {
		t.Fatalf("WriteStructured() should omit empty module, got %q", out)
	}
}
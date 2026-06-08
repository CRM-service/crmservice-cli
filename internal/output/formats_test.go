package output

import "testing"

func TestOutputFormatsCanonicalList(t *testing.T) {
	expected := []string{"table", "json", "yaml", "jsonl", "csv"}
	if len(OutputFormats) != len(expected) {
		t.Fatalf("OutputFormats = %v, want %v", OutputFormats, expected)
	}
	for i, format := range expected {
		if OutputFormats[i] != format {
			t.Fatalf("OutputFormats[%d] = %q, want %q", i, OutputFormats[i], format)
		}
	}
}

func TestValidateOutputFormat(t *testing.T) {
	if err := ValidateOutputFormat("json"); err != nil {
		t.Fatalf("ValidateOutputFormat(json) error = %v", err)
	}
	if err := ValidateOutputFormat("xml"); err == nil {
		t.Fatal("ValidateOutputFormat(xml) error = nil, expected error")
	}
}

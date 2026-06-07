package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestEmitUsageErrorPrintsCobraStyleUsage(t *testing.T) {
	cmd := listCmd()
	stderr := captureStderr(t, func() {
		emitUsageError(cmd, errors.New("accepts 1 arg(s), received 0"))
	})

	if !strings.Contains(stderr, "Error: accepts 1 arg(s), received 0") {
		t.Fatalf("stderr missing error: %q", stderr)
	}
	if !strings.Contains(stderr, "Usage:") || !strings.Contains(stderr, "list <module>") {
		t.Fatalf("stderr missing usage: %q", stderr)
	}
	if strings.Contains(stderr, `"error":true`) || strings.Contains(stderr, "message              |") {
		t.Fatalf("usage errors should not use structured error formatting: %q", stderr)
	}
}

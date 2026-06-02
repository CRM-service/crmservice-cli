package cmd

import "testing"

func TestVersionStringDefaultsToDev(t *testing.T) {
	oldVersion := version
	version = "dev"
	t.Cleanup(func() { version = oldVersion })

	if versionString() != "dev" {
		t.Errorf("versionString() = %q, expected dev", versionString())
	}
}

func TestVersionStringUsesInjectedVersion(t *testing.T) {
	oldVersion := version
	version = "v1.2.3"
	t.Cleanup(func() { version = oldVersion })

	if versionString() != "v1.2.3" {
		t.Errorf("versionString() = %q, expected v1.2.3", versionString())
	}
}

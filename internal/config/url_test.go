package config

import "testing"

func TestResolveAPIURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		require bool
		want    string
		wantErr bool
	}{
		{name: "hostname", raw: "customer.crmservice.fi", require: true, want: "https://customer.crmservice.fi/api/v1"},
		{name: "https base", raw: "https://customer.crmservice.fi/api/v1", require: true, want: "https://customer.crmservice.fi/api/v1"},
		{name: "optional empty", raw: "", require: false, want: ""},
		{name: "required empty", raw: "", require: true, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveAPIURL(tc.raw, tc.require)
			if tc.wantErr {
				if err == nil {
					t.Fatal("ResolveAPIURL() error = nil, expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveAPIURL() returned error: %v", err)
			}
			if got != tc.want {
				t.Errorf("ResolveAPIURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDisplayAPIURL(t *testing.T) {
	got := DisplayAPIURL("customer.crmservice.fi")
	want := "https://customer.crmservice.fi/api/v1"
	if got != want {
		t.Errorf("DisplayAPIURL() = %q, want %q", got, want)
	}
}

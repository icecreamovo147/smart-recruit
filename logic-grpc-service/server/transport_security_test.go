package server

import "testing"

func TestTransportSecurityOptionDefaultsToDisabled(t *testing.T) {
	option, enabled, err := TransportSecurityOption("", "")
	if err != nil {
		t.Fatalf("TransportSecurityOption: %v", err)
	}
	if enabled {
		t.Fatal("enabled = true, want false")
	}
	if option != nil {
		t.Fatal("option should be nil when TLS is disabled")
	}
}

func TestTransportSecurityOptionRequiresCertAndKey(t *testing.T) {
	_, _, err := TransportSecurityOption("/etc/recruitment/tls/tls.crt", "")
	if err == nil {
		t.Fatal("expected missing key error")
	}
}

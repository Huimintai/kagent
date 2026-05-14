package sandboxbackend

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAllowedDomainHost(t *testing.T) {
	tests := []struct {
		raw  string
		want string
		ok   bool
	}{
		{"api.openai.com", "api.openai.com", true},
		{"  *.anthropic.com  ", "*.anthropic.com", true},
		{"https://api.telegram.org/bot", "api.telegram.org", true},
		{"http://example.com:8080/path", "example.com", true},
		{"host.only:443", "host.only", true},
		{"", "", false},
		{"https://", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, ok := NormalizeAllowedDomainHost(tt.raw)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeAllowedDomainEntry(t *testing.T) {
	tests := []struct {
		raw      string
		wantHost string
		wantPort uint32
		ok       bool
	}{
		// no port → port 0
		{"api.openai.com", "api.openai.com", 0, true},
		{"https://api.openai.com/v1", "api.openai.com", 0, true},
		// explicit standard ports
		{"host.only:443", "host.only", 443, true},
		{"http://example.com:80/path", "example.com", 80, true},
		// custom non-standard port (the main use-case: cluster-internal HTTP service)
		{"sap-ai-proxy.kagent.svc.cluster.local:3030", "sap-ai-proxy.kagent.svc.cluster.local", 3030, true},
		{"http://sap-ai-proxy.kagent.svc.cluster.local:3030/v1", "sap-ai-proxy.kagent.svc.cluster.local", 3030, true},
		// URL with non-standard port and path
		{"http://example.com:8080/path", "example.com", 8080, true},
		// invalid
		{"", "", 0, false},
		{"https://", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			gotHost, gotPort, ok := NormalizeAllowedDomainEntry(tt.raw)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.wantHost, gotHost)
			require.Equal(t, tt.wantPort, gotPort)
		})
	}
}

package run

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestIPExtractorTrust(t *testing.T) {
	logger := zerolog.Nop()

	newReq := func(remoteAddr string, headers map[string]string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", nil)
		r.RemoteAddr = remoteAddr
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		return r
	}

	tests := []struct {
		name           string
		trustPrivate   bool
		trustedProxies []string
		remoteAddr     string
		headers        map[string]string
		want           string
	}{
		{
			name:         "public peer cannot spoof X-Forwarded-For",
			trustPrivate: true,
			remoteAddr:   "203.0.113.7:54321",
			headers:      map[string]string{"X-Forwarded-For": "1.2.3.4"},
			want:         "203.0.113.7",
		},
		{
			name:         "public peer cannot spoof X-Real-IP",
			trustPrivate: true,
			remoteAddr:   "203.0.113.7:54321",
			headers:      map[string]string{"X-Real-IP": "1.2.3.4"},
			want:         "203.0.113.7",
		},
		{
			name:         "trusted private proxy XFF is honored",
			trustPrivate: true,
			remoteAddr:   "10.0.0.5:443",
			headers:      map[string]string{"X-Forwarded-For": "1.2.3.4"},
			want:         "1.2.3.4",
		},
		{
			name:         "spoofed left-most XFF entry is ignored, real client returned",
			trustPrivate: true,
			remoteAddr:   "10.0.0.5:443",
			headers:      map[string]string{"X-Forwarded-For": "9.9.9.9, 1.2.3.4"},
			want:         "1.2.3.4",
		},
		{
			name:           "explicitly trusted proxy CIDR honors XFF",
			trustPrivate:   false,
			trustedProxies: []string{"198.51.100.0/24"},
			remoteAddr:     "198.51.100.10:443",
			headers:        map[string]string{"X-Forwarded-For": "1.2.3.4"},
			want:           "1.2.3.4",
		},
		{
			name:           "peer outside trusted CIDR cannot spoof",
			trustPrivate:   false,
			trustedProxies: []string{"198.51.100.0/24"},
			remoteAddr:     "203.0.113.7:54321",
			headers:        map[string]string{"X-Forwarded-For": "1.2.3.4"},
			want:           "203.0.113.7",
		},
		{
			name:         "trust nothing ignores all headers",
			trustPrivate: false,
			remoteAddr:   "203.0.113.7:54321",
			headers:      map[string]string{"X-Forwarded-For": "1.2.3.4", "X-Real-IP": "5.6.7.8"},
			want:         "203.0.113.7",
		},
		{
			name:         "no headers falls back to socket address",
			trustPrivate: true,
			remoteAddr:   "203.0.113.7:54321",
			want:         "203.0.113.7",
		},
		{
			name:           "invalid trusted CIDR is ignored, private trust still applies",
			trustPrivate:   true,
			trustedProxies: []string{"not-a-cidr"},
			remoteAddr:     "10.0.0.5:443",
			headers:        map[string]string{"X-Forwarded-For": "1.2.3.4"},
			want:           "1.2.3.4",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			extractor := hatchetIPExtractor(tc.trustPrivate, tc.trustedProxies, &logger)
			got := extractor(newReq(tc.remoteAddr, tc.headers))
			if got != tc.want {
				t.Fatalf("extractor returned %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIPExtractorRateLimitIdentityStable(t *testing.T) {
	logger := zerolog.Nop()
	extractor := hatchetIPExtractor(true, nil, &logger)

	spoofed := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}
	first := ""

	for _, ip := range spoofed {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", nil)
		r.RemoteAddr = "203.0.113.7:54321"
		r.Header.Set("CF-Connecting-IP", ip)
		r.Header.Set("X-Forwarded-For", ip)

		got := extractor(r)
		if first == "" {
			first = got
		}
		if got != first {
			t.Fatalf("identity changed to %q when rotating spoofed header; bypass not closed", got)
		}
	}

	if first != "203.0.113.7" {
		t.Fatalf("expected identity to be the socket IP 203.0.113.7, got %q", first)
	}
}

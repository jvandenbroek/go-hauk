package ratelimit

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		trustProxy bool
		remoteAddr string
		xff        string
		want       string
	}{
		{"remote ipv4", false, "192.0.2.1:1234", "", "192.0.2.1"},
		{"remote ipv6", false, "[2001:db8::1]:1234", "", "2001:db8::1"},
		{"xff ignored when untrusted", false, "192.0.2.1:1234", "198.51.100.7", "192.0.2.1"},
		{"xff single", true, "10.0.0.1:1234", "198.51.100.7", "198.51.100.7"},
		{"xff spoofed prefix", true, "10.0.0.1:1234", "1.2.3.4, 198.51.100.7", "198.51.100.7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(1, time.Minute, tt.trustProxy)
			r := httptest.NewRequest("POST", "/", nil)
			r.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if got := l.clientIP(r); got != tt.want {
				t.Errorf("clientIP = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAllowLimit(t *testing.T) {
	l := New(2, time.Minute, false)
	if !l.Allow("a") || !l.Allow("a") {
		t.Fatal("first two requests should pass")
	}
	if l.Allow("a") {
		t.Error("third request should be limited")
	}
	if !l.Allow("b") {
		t.Error("other key should not be limited")
	}
}

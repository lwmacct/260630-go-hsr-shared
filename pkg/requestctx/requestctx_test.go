package requestctx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareUsesForwardedIPFromTrustedProxy(t *testing.T) {
	middleware := NewMiddleware([]string{"10.0.0.1"})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.1")

	request, ok := middleware.Request(req)
	if !ok {
		t.Fatal("request not resolved")
	}
	if request.IP != "203.0.113.10" {
		t.Fatalf("request IP = %q", request.IP)
	}
}

func TestMiddlewareIgnoresForwardedIPFromUntrustedProxy(t *testing.T) {
	middleware := NewMiddleware([]string{"10.0.0.1"})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.10")

	request, ok := middleware.Request(req)
	if !ok {
		t.Fatal("request not resolved")
	}
	if request.IP != "10.0.0.2" {
		t.Fatalf("request IP = %q", request.IP)
	}
}

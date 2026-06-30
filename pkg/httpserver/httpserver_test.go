package httpserver

import (
	"net/http"
	"testing"
)

func TestShouldLimitRequestBody(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/api", http.NoBody)
	if ShouldLimitRequestBody(req) {
		t.Fatal("empty body should not be limited")
	}
	req, _ = http.NewRequest(http.MethodPost, "/api", &body{})
	if !ShouldLimitRequestBody(req) {
		t.Fatal("post body should be limited")
	}
	req.Header.Set("Upgrade", "websocket")
	if ShouldLimitRequestBody(req) {
		t.Fatal("websocket upgrade should not be limited")
	}
}

type body struct{}

func (*body) Read([]byte) (int, error) {
	return 0, nil
}

func (*body) Close() error {
	return nil
}

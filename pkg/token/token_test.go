package token

import (
	"strings"
	"testing"
)

func TestNewWithPrefix(t *testing.T) {
	value, err := NewWithPrefix("cap")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(value, "cap_") {
		t.Fatalf("token prefix = %q", value)
	}
	if len(value) <= len("cap_") {
		t.Fatalf("token too short: %q", value)
	}
}

func TestNewBase62RequiresPositiveLength(t *testing.T) {
	if _, err := NewBase62(0); err == nil {
		t.Fatal("expected error")
	}
}

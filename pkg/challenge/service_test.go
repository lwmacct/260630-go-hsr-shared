package challenge

import (
	"context"
	"errors"
	"testing"
)

func TestServiceRejectsWrongProvider(t *testing.T) {
	service := NewService(passProvider{})
	err := service.Verify(context.Background(), Answer{Provider: "other"}, Input{})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestServiceWrapsProviderVerifyError(t *testing.T) {
	service := NewService(failProvider{})
	err := service.Verify(context.Background(), Answer{Provider: "fail"}, Input{})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

type passProvider struct{}

func (passProvider) Name() string {
	return "pass"
}

func (passProvider) PublicConfig() PublicConfig {
	return PublicConfig{Provider: "pass"}
}

func (passProvider) Create(context.Context, Input) (*Challenge, error) {
	return &Challenge{Provider: "pass"}, nil
}

func (passProvider) Verify(context.Context, Answer, Input) error {
	return nil
}

type failProvider struct{}

func (failProvider) Name() string {
	return "fail"
}

func (failProvider) PublicConfig() PublicConfig {
	return PublicConfig{Provider: "fail"}
}

func (failProvider) Create(context.Context, Input) (*Challenge, error) {
	return &Challenge{Provider: "fail"}, nil
}

func (failProvider) Verify(context.Context, Answer, Input) error {
	return errors.New("provider failed")
}

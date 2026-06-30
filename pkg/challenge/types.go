package challenge

import (
	"context"
	"errors"
	"time"
)

const (
	ProviderImage     = "image"
	ProviderHCaptcha  = "hcaptcha"
	ProviderTurnstile = "turnstile"
)

var (
	ErrInvalid       = errors.New("invalid challenge")
	ErrUnsupported   = errors.New("challenge provider unsupported")
	ErrLimitExceeded = errors.New("challenge limit exceeded")
)

type Input struct {
	IP         string
	UserAgent  string
	Method     string
	Path       string
	RemoteAddr string
}

type PublicConfig struct {
	Provider string
	SiteKey  string
}

type Challenge struct {
	Provider    string
	ChallengeID string
	Image       string
	ExpiresAt   time.Time
}

type Answer struct {
	Provider    string
	ChallengeID string
	Answer      string
	Token       string
}

type Provider interface {
	Name() string
	PublicConfig() PublicConfig
	Create(context.Context, Input) (*Challenge, error)
	Verify(context.Context, Answer, Input) error
}

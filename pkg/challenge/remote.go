package challenge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RemoteTokenProvider struct {
	provider  string
	siteKey   string
	secret    string
	verifyURL string
	client    *http.Client
}

func NewRemoteTokenProvider(provider string, siteKey string, secret string, verifyURL string) (*RemoteTokenProvider, error) {
	if strings.TrimSpace(provider) == "" || strings.TrimSpace(siteKey) == "" || strings.TrimSpace(secret) == "" || strings.TrimSpace(verifyURL) == "" {
		return nil, ErrUnsupported
	}
	return &RemoteTokenProvider{
		provider:  strings.TrimSpace(provider),
		siteKey:   strings.TrimSpace(siteKey),
		secret:    strings.TrimSpace(secret),
		verifyURL: strings.TrimSpace(verifyURL),
		client:    &http.Client{Timeout: 5 * time.Second},
	}, nil
}

func (p *RemoteTokenProvider) Name() string {
	return p.provider
}

func (p *RemoteTokenProvider) PublicConfig() PublicConfig {
	return PublicConfig{Provider: p.provider, SiteKey: p.siteKey}
}

func (p *RemoteTokenProvider) Create(context.Context, Input) (*Challenge, error) {
	return nil, ErrUnsupported
}

func (p *RemoteTokenProvider) Verify(ctx context.Context, response Answer, request Input) error {
	if strings.TrimSpace(response.Token) == "" {
		return ErrInvalid
	}
	form := url.Values{}
	form.Set("secret", p.secret)
	form.Set("response", response.Token)
	if request.IP != "" {
		form.Set("remoteip", request.IP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ErrInvalid
	}
	var body struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}
	if !body.Success {
		return ErrInvalid
	}
	return nil
}

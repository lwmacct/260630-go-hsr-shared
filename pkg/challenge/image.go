package challenge

import (
	"context"
	"crypto/sha256"
	"errors"
	"image/color"
	"strings"
	"sync"
	"time"

	"github.com/golang-module/base64Captcha/driver"

	"github.com/lwmacct/260630-go-hsr-shared/pkg/token"
)

const imageDefaultTTL = 2 * time.Minute

type ImageProvider struct {
	mu         sync.Mutex
	challenges map[string]imageChallenge
	driver     driver.Driver
	ttl        time.Duration
	maxItems   int
}

type imageChallenge struct {
	answerHash [32]byte
	expiresAt  time.Time
}

func NewImageProvider(maxItems int) *ImageProvider {
	return &ImageProvider{
		challenges: make(map[string]imageChallenge),
		driver: driver.NewDriverString(driver.DriverString{
			Width:           180,
			Height:          56,
			Length:          4,
			NoiseCount:      12,
			ShowLineOptions: driver.OptionShowHollowLine | driver.OptionShowSlimeLine,
			Source:          "23456789ABCDEFGHJKLMNPQRSTUVWXYZ",
			BgColor:         &color.RGBA{R: 248, G: 250, B: 252, A: 255},
		}),
		ttl:      imageDefaultTTL,
		maxItems: maxItems,
	}
}

func (p *ImageProvider) Name() string {
	return ProviderImage
}

func (p *ImageProvider) PublicConfig() PublicConfig {
	return PublicConfig{Provider: ProviderImage}
}

func (p *ImageProvider) Create(context.Context, Input) (*Challenge, error) {
	_, content, answer := p.driver.GenerateCaptcha()
	image, err := p.driver.DrawCaptcha(content)
	if err != nil {
		return nil, err
	}
	id, expiresAt, err := p.put(answer)
	if err != nil {
		if errors.Is(err, ErrLimitExceeded) {
			return nil, ErrLimitExceeded
		}
		return nil, err
	}
	return &Challenge{
		Provider:    ProviderImage,
		ChallengeID: id,
		Image:       image.Encoder(),
		ExpiresAt:   expiresAt,
	}, nil
}

func (p *ImageProvider) Verify(_ context.Context, response Answer, _ Input) error {
	if !p.verifyAndDelete(response.ChallengeID, response.Answer) {
		return ErrInvalid
	}
	return nil
}

func (p *ImageProvider) put(answer string) (string, time.Time, error) {
	id, err := token.NewWithPrefix("cap")
	if err != nil {
		return "", time.Time{}, err
	}
	now := time.Now().UTC()
	expiresAt := now.Add(p.ttl)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cleanupLocked(now)
	if p.maxItems > 0 && len(p.challenges) >= p.maxItems {
		return "", time.Time{}, ErrLimitExceeded
	}
	p.challenges[id] = imageChallenge{answerHash: imageAnswerHash(answer), expiresAt: expiresAt}
	return id, expiresAt, nil
}

func (p *ImageProvider) verifyAndDelete(id string, answer string) bool {
	id = strings.TrimSpace(id)
	if id == "" || strings.TrimSpace(answer) == "" {
		return false
	}
	now := time.Now().UTC()
	p.mu.Lock()
	defer p.mu.Unlock()
	challenge, ok := p.challenges[id]
	if !ok {
		return false
	}
	delete(p.challenges, id)
	if !challenge.expiresAt.After(now) {
		return false
	}
	return challenge.answerHash == imageAnswerHash(answer)
}

func (p *ImageProvider) cleanupLocked(now time.Time) {
	for id, challenge := range p.challenges {
		if !challenge.expiresAt.After(now) {
			delete(p.challenges, id)
		}
	}
}

func imageAnswerHash(answer string) [32]byte {
	return sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(answer))))
}

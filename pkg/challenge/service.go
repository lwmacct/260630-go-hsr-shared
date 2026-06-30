package challenge

import "context"

type Service struct {
	provider Provider
}

func NewService(provider Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) PublicConfig() PublicConfig {
	if s == nil || s.provider == nil {
		return PublicConfig{}
	}
	return s.provider.PublicConfig()
}

func (s *Service) Create(ctx context.Context, input Input) (*Challenge, error) {
	if s == nil || s.provider == nil {
		return nil, ErrUnsupported
	}
	return s.provider.Create(ctx, input)
}

func (s *Service) Verify(ctx context.Context, answer Answer, input Input) error {
	if s == nil || s.provider == nil || answer.Provider != s.provider.Name() {
		return ErrInvalid
	}
	if err := s.provider.Verify(ctx, answer, input); err != nil {
		return ErrInvalid
	}
	return nil
}

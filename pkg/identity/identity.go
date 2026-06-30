package identity

import (
	"context"
	"time"

	"github.com/lwmacct/260630-go-hsr-shared/pkg/requestctx"
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type Principal struct {
	ID          string
	Subject     string
	Username    string
	DisplayName string
	Email       string
	AvatarURL   string
	Role        string
	Status      string
	Admin       bool
	DisabledAt  *time.Time
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (p *Principal) Active() bool {
	return p != nil && p.ID != "" && p.Status != StatusDisabled && p.DisabledAt == nil
}

type SessionResolver interface {
	CurrentPrincipal(context.Context, string, requestctx.Request) (*Principal, error)
}

type Directory interface {
	Principal(context.Context, string) (*Principal, error)
	Principals(context.Context, []string) (map[string]*Principal, error)
}

package appmodule_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lwmacct/260630-go-hsr-shared/pkg/appmodule"
	"github.com/uptrace/bun"
)

func TestApplySchemasWrapsModuleName(t *testing.T) {
	err := appmodule.ApplySchemas(context.Background(), nil, testModule{name: "auth", schemaErr: errors.New("boom")})
	if err == nil || !strings.Contains(err.Error(), "apply auth schema") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitWrapsModuleName(t *testing.T) {
	err := appmodule.Init(context.Background(), nil, testModule{name: "oauth", initErr: errors.New("boom")})
	if err == nil || !strings.Contains(err.Error(), "configure oauth module") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type testModule struct {
	name      string
	schemaErr error
	initErr   error
}

func (m testModule) Name() string {
	return m.name
}

func (m testModule) ApplySchema(context.Context, *bun.DB) error {
	return m.schemaErr
}

func (m testModule) Init(context.Context, *bun.DB) error {
	return m.initErr
}

func (m testModule) Register(huma.API) {}

var _ appmodule.Module = testModule{}

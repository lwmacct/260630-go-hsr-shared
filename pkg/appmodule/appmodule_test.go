package appmodule_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lwmacct/260630-go-hsr-shared/pkg/appmodule"
	"github.com/uptrace/bun"
)

func TestBuildSortsByDependencies(t *testing.T) {
	var calls []string
	runtime, err := appmodule.Build(context.Background(), nil,
		spec("audit", []string{"auth"}, &calls),
		spec("auth", nil, &calls),
		spec("oauth", []string{"auth"}, &calls),
	)
	if err != nil {
		t.Fatal(err)
	}

	if got := names(runtime.Modules()); !reflect.DeepEqual(got, []string{"auth", "audit", "oauth"}) {
		t.Fatalf("order = %v", got)
	}
	if !reflect.DeepEqual(calls, []string{
		"schema:auth", "build:auth",
		"schema:audit", "build:audit",
		"schema:oauth", "build:oauth",
	}) {
		t.Fatalf("calls = %v", calls)
	}
}

func TestBuildRejectsMissingDependency(t *testing.T) {
	_, err := appmodule.Build(context.Background(), nil, spec("oauth", []string{"auth"}, nil))
	if err == nil || !strings.Contains(err.Error(), "requires missing module auth") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsDependencyCycle(t *testing.T) {
	_, err := appmodule.Build(context.Background(), nil,
		spec("auth", []string{"audit"}, nil),
		spec("audit", []string{"auth"}, nil),
	)
	if err == nil || !strings.Contains(err.Error(), "module dependency cycle") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildWrapsSchemaError(t *testing.T) {
	_, err := appmodule.Build(context.Background(), nil, appmodule.Spec{
		Name: "auth",
		Schema: func(context.Context, *bun.DB) error {
			return errors.New("boom")
		},
		Build: func(*appmodule.Context) (appmodule.Module, error) {
			return testModule{name: "auth"}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "apply auth schema") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildWrapsBuildError(t *testing.T) {
	_, err := appmodule.Build(context.Background(), nil, appmodule.Spec{
		Name:   "auth",
		Schema: func(context.Context, *bun.DB) error { return nil },
		Build: func(*appmodule.Context) (appmodule.Module, error) {
			return nil, errors.New("boom")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "build auth module") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildClosesInitializedModulesOnFailure(t *testing.T) {
	var closed []string
	_, err := appmodule.Build(context.Background(), nil,
		appmodule.Spec{
			Name:   "auth",
			Schema: func(context.Context, *bun.DB) error { return nil },
			Build: func(*appmodule.Context) (appmodule.Module, error) {
				return closeModule{name: "auth", closed: &closed}, nil
			},
		},
		appmodule.Spec{
			Name:     "oauth",
			Requires: []string{"auth"},
			Schema:   func(context.Context, *bun.DB) error { return errors.New("boom") },
			Build: func(*appmodule.Context) (appmodule.Module, error) {
				return testModule{name: "oauth"}, nil
			},
		},
	)
	if err == nil {
		t.Fatal("expected build failure")
	}
	if !reflect.DeepEqual(closed, []string{"auth"}) {
		t.Fatalf("closed = %v", closed)
	}
}

func TestMustContextGetReturnsDependency(t *testing.T) {
	runtime, err := appmodule.Build(context.Background(), nil, spec("auth", nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	auth := appmodule.MustGet[testModule](runtime, "auth")
	if auth.Name() != "auth" {
		t.Fatalf("unexpected module: %v", auth.Name())
	}
}

func spec(name string, requires []string, calls *[]string) appmodule.Spec {
	return appmodule.Spec{
		Name:     name,
		Requires: requires,
		Schema: func(context.Context, *bun.DB) error {
			if calls != nil {
				*calls = append(*calls, "schema:"+name)
			}
			return nil
		},
		Build: func(*appmodule.Context) (appmodule.Module, error) {
			if calls != nil {
				*calls = append(*calls, "build:"+name)
			}
			return testModule{name: name}, nil
		},
	}
}

func names(modules []appmodule.Module) []string {
	names := make([]string, 0, len(modules))
	for _, module := range modules {
		names = append(names, module.Name())
	}
	return names
}

type testModule struct {
	name string
}

func (m testModule) Name() string {
	return m.name
}

func (m testModule) Register(huma.API) {}

type closeModule struct {
	name   string
	closed *[]string
}

func (m closeModule) Name() string {
	return m.name
}

func (m closeModule) Register(huma.API) {}

func (m closeModule) Close() error {
	*m.closed = append(*m.closed, m.name)
	return nil
}

var _ appmodule.Module = testModule{}

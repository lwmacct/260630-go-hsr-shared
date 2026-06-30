package appmodule

import (
	"context"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/uptrace/bun"
)

type Named interface {
	Name() string
}

type SchemaApplier interface {
	Named
	ApplySchema(context.Context, *bun.DB) error
}

type Initializer interface {
	Named
	Init(context.Context, *bun.DB) error
}

type Registrar interface {
	Named
	Register(huma.API)
}

type Module interface {
	SchemaApplier
	Initializer
	Registrar
}

func ApplySchemas(ctx context.Context, db *bun.DB, modules ...SchemaApplier) error {
	for _, module := range modules {
		if err := module.ApplySchema(ctx, db); err != nil {
			return fmt.Errorf("apply %s schema: %w", module.Name(), err)
		}
	}
	return nil
}

func Init(ctx context.Context, db *bun.DB, modules ...Initializer) error {
	for _, module := range modules {
		if err := module.Init(ctx, db); err != nil {
			return fmt.Errorf("configure %s module: %w", module.Name(), err)
		}
	}
	return nil
}

func Register(api huma.API, modules ...Registrar) {
	for _, module := range modules {
		module.Register(api)
	}
}

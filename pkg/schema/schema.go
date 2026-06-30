package schema

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func Apply(ctx context.Context, db *bun.DB, models ...any) error {
	for _, model := range models {
		if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}
	return nil
}

func Reset(ctx context.Context, db *bun.DB, tables []string, models ...any) error {
	for _, table := range tables {
		if _, err := db.NewDropTable().Table(table).IfExists().Cascade().Exec(ctx); err != nil {
			return fmt.Errorf("drop table %s: %w", table, err)
		}
	}
	return Apply(ctx, db, models...)
}

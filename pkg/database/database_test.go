package database

import (
	"context"
	"strings"
	"testing"
)

func TestOpenSQLiteMemory(t *testing.T) {
	db, err := OpenSQLite(context.Background(), "file:shared-db-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
}

func TestBuildPGSQLDSNDefaults(t *testing.T) {
	dsn := BuildPGSQLDSN(PGSQLConfig{})
	for _, value := range []string{"postgres://postgres@localhost:5432/postgres", "sslmode=disable"} {
		if !strings.Contains(dsn, value) {
			t.Fatalf("dsn %q does not contain %q", dsn, value)
		}
	}
}

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
)

type Config struct {
	Type   string
	SQLite string
	PGSQL  PGSQLConfig
}

type PGSQLConfig struct {
	Host     string
	Port     string
	User     string
	Database string
	Password string
}

func Open(ctx context.Context, cfg Config) (*bun.DB, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Type)) {
	case "", "sqlite":
		return OpenSQLite(ctx, cfg.SQLite)
	case "pgsql":
		return OpenPGSQL(ctx, cfg.PGSQL)
	default:
		return nil, fmt.Errorf("unsupported database type %q", cfg.Type)
	}
}

func OpenSQLite(ctx context.Context, path string) (*bun.DB, error) {
	if path == "" {
		return nil, errors.New("database path is required")
	}
	if path != ":memory:" && !strings.HasPrefix(path, "file:") {
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}
	dsn := path
	if strings.Contains(dsn, "?") {
		dsn += "&_pragma=foreign_keys(1)"
	} else {
		dsn += "?_pragma=foreign_keys(1)"
	}
	sqlDB, err := sql.Open(sqliteshim.ShimName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}
	return bun.NewDB(sqlDB, sqlitedialect.New()), nil
}

func OpenPGSQL(ctx context.Context, cfg PGSQLConfig) (*bun.DB, error) {
	sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(BuildPGSQLDSN(cfg))))
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping pgsql database: %w", err)
	}
	return bun.NewDB(sqlDB, pgdialect.New()), nil
}

func BuildPGSQLDSN(cfg PGSQLConfig) string {
	user := configValue(cfg.User, "PGUSER", "postgres")
	password := configValue(cfg.Password, "PGPASSWORD", "")
	database := configValue(cfg.Database, "PGDATABASE", user)
	host := configValue(cfg.Host, "PGHOST", "localhost")
	port := configValue(cfg.Port, "PGPORT", "5432")

	dsn := url.URL{Scheme: "postgres", User: url.User(user), Path: "/" + database}
	if password != "" {
		dsn.User = url.UserPassword(user, password)
	}
	if strings.HasPrefix(host, "/") {
		dsn.Host = "localhost"
		query := dsn.Query()
		query.Set("host", host)
		dsn.RawQuery = query.Encode()
	} else {
		dsn.Host = net.JoinHostPort(host, port)
	}
	query := dsn.Query()
	if query.Get("sslmode") == "" {
		query.Set("sslmode", "disable")
	}
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func configValue(value string, envKey string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "${"+envKey+"}" {
		if envValue := strings.TrimSpace(os.Getenv(envKey)); envValue != "" {
			return envValue
		}
		return fallback
	}
	return trimmed
}

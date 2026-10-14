package migrate

import (
	"os"
	"testing"
)

func TestResolveDatabaseURL(t *testing.T) {
	origURL := os.Getenv("DATABASE_URL")
	origHost := os.Getenv("DATABASE_POSTGRES_HOST")
	origPort := os.Getenv("DATABASE_POSTGRES_PORT")
	origUser := os.Getenv("DATABASE_POSTGRES_USERNAME")
	origPass := os.Getenv("DATABASE_POSTGRES_PASSWORD")
	origDb := os.Getenv("DATABASE_POSTGRES_DB_NAME")
	origSSL := os.Getenv("DATABASE_POSTGRES_SSL_MODE")

	defer func() {
		os.Setenv("DATABASE_URL", origURL)
		os.Setenv("DATABASE_POSTGRES_HOST", origHost)
		os.Setenv("DATABASE_POSTGRES_PORT", origPort)
		os.Setenv("DATABASE_POSTGRES_USERNAME", origUser)
		os.Setenv("DATABASE_POSTGRES_PASSWORD", origPass)
		os.Setenv("DATABASE_POSTGRES_DB_NAME", origDb)
		os.Setenv("DATABASE_POSTGRES_SSL_MODE", origSSL)
	}()

	t.Run("configured URL takes precedence", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgresql://env:env@env:5432/env")
		got := resolveDatabaseURL("postgresql://conf:conf@conf:5432/conf")
		if got != "postgresql://conf:conf@conf:5432/conf" {
			t.Errorf("got %q, want configured URL", got)
		}
	})

	t.Run("DATABASE_URL env takes precedence over POSTGRES_* vars", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgresql://env:env@env:5432/env")
		os.Setenv("DATABASE_POSTGRES_HOST", "pg-host")
		got := resolveDatabaseURL("")
		if got != "postgresql://env:env@env:5432/env" {
			t.Errorf("got %q, want env DATABASE_URL", got)
		}
	})

	t.Run("fallback to DATABASE_POSTGRES_* vars when DATABASE_URL is unset", func(t *testing.T) {
		os.Unsetenv("DATABASE_URL")
		os.Setenv("DATABASE_POSTGRES_HOST", "my-postgres")
		os.Setenv("DATABASE_POSTGRES_PORT", "5432")
		os.Setenv("DATABASE_POSTGRES_USERNAME", "myuser")
		os.Setenv("DATABASE_POSTGRES_PASSWORD", "mypass")
		os.Setenv("DATABASE_POSTGRES_DB_NAME", "mydb")
		os.Setenv("DATABASE_POSTGRES_SSL_MODE", "require")

		want := "postgresql://myuser:mypass@my-postgres:5432/mydb?sslmode=require"
		got := resolveDatabaseURL("")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("defaults applied when partial POSTGRES_* vars set", func(t *testing.T) {
		os.Unsetenv("DATABASE_URL")
		os.Setenv("DATABASE_POSTGRES_HOST", "my-postgres")
		os.Unsetenv("DATABASE_POSTGRES_PORT")
		os.Unsetenv("DATABASE_POSTGRES_USERNAME")
		os.Unsetenv("DATABASE_POSTGRES_PASSWORD")
		os.Unsetenv("DATABASE_POSTGRES_DB_NAME")
		os.Unsetenv("DATABASE_POSTGRES_SSL_MODE")

		want := "postgresql://hatchet:hatchet@my-postgres:5431/hatchet?sslmode=disable"
		got := resolveDatabaseURL("")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("returns empty string when no vars set", func(t *testing.T) {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("DATABASE_POSTGRES_HOST")
		os.Unsetenv("DATABASE_POSTGRES_PORT")
		os.Unsetenv("DATABASE_POSTGRES_USERNAME")
		os.Unsetenv("DATABASE_POSTGRES_PASSWORD")
		os.Unsetenv("DATABASE_POSTGRES_DB_NAME")
		os.Unsetenv("DATABASE_POSTGRES_SSL_MODE")

		got := resolveDatabaseURL("")
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})
}

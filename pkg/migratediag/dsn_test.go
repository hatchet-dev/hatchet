package migratediag

import (
	"errors"
	"strings"
	"testing"
)

func TestSummarizePostgresDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "<empty DSN>",
		},
		{
			name: "url form with creds, query and tls",
			in:   "postgres://alice:s3cret@db.example.com:5432/mydb?sslmode=require&application_name=hatchet-migrate&channel_binding=prefer",
			want: "postgres://<redacted>@db.example.com:5432/mydb?keys=application_name,tls",
		},
		{
			name: "url form with sslmode=disable",
			in:   "postgres://alice:s3cret@db.example.com:5432/mydb?sslmode=disable",
			want: "postgres://<redacted>@db.example.com:5432/mydb",
		},
		{
			name: "libpq keyword form with password",
			in:   "host=db.example.com port=5432 user=alice password=hunter2 dbname=mydb sslmode=require application_name=hatchet-migrate",
			want: "postgres://<redacted>@db.example.com:5432/mydb?keys=application_name,tls",
		},
		{
			name: "libpq keyword form without port",
			in:   "host=/var/run/postgresql user=alice password=hunter2 dbname=mydb sslmode=disable",
			want: "postgres://<redacted>@/var/run/postgresql:5432/mydb",
		},
		{
			name: "unparseable",
			in:   "postgres://user:p%ZZword@host/db",
			want: "<unparseable DSN>",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SummarizePostgresDSN(tc.in)
			if string(got) != tc.want {
				t.Fatalf("SummarizePostgresDSN(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSummarizePostgresDSN_NeverLeaksSecrets(t *testing.T) {
	t.Parallel()

	const (
		username = "alice"
		password = "s3cret-do-not-log"
		appName  = "very-specific-app-name-value"
	)

	cases := []string{
		"postgres://" + username + ":" + password + "@db.example.com:5432/mydb?application_name=" + appName + "&sslmode=require",
		"host=db.example.com port=5432 user=" + username + " password=" + password + " dbname=mydb application_name=" + appName + " sslmode=require",
	}

	for _, in := range cases {
		got := SummarizePostgresDSN(in)
		for _, leak := range []string{username, password, appName} {
			if strings.Contains(string(got), leak) {
				t.Fatalf("SummarizePostgresDSN leaked %q in %q (input: %q)", leak, got, in)
			}
		}
	}
}

func TestSummarizeURLDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "<empty DSN>",
		},
		{
			name: "url with username, password and query params",
			in:   "clickhouse://alice:s3cret@db.example.com:9000/mydb?sslmode=require&channel_binding=prefer",
			want: "clickhouse://<redacted>@db.example.com:9000/mydb?keys=channel_binding,sslmode",
		},
		{
			name: "url without credentials",
			in:   "clickhouse://db.example.com:9000/mydb",
			want: "clickhouse://db.example.com:9000/mydb",
		},
		{
			name: "url without query string",
			in:   "clickhouse://alice:s3cret@db.example.com:9000/mydb",
			want: "clickhouse://<redacted>@db.example.com:9000/mydb",
		},
		{
			name: "clickhousess url with credentials and query",
			in:   "clickhousess://user:hunter2@ch.example.com:9440/analytics?compress=true&secure=true",
			want: "clickhousess://<redacted>@ch.example.com:9440/analytics?keys=compress,secure",
		},
		{
			name: "host without port",
			in:   "clickhouse://user:pw@db.example.com/mydb",
			want: "clickhouse://<redacted>@db.example.com/mydb",
		},
		{
			name: "unparseable url",
			in:   "clickhouse://user:p%ZZword@host/db",
			want: "<unparseable DSN>",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SummarizeURLDSN(tc.in)
			if string(got) != tc.want {
				t.Fatalf("SummarizeURLDSN(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSummarizeURLDSN_NeverLeaksSecrets(t *testing.T) {
	t.Parallel()

	const (
		username  = "alice"
		password  = "s3cret-do-not-log"
		queryVal  = "require-tls-1.3-only"
		queryVal2 = "prefer-channel-binding"
	)

	in := "clickhousess://" + username + ":" + password +
		"@db.example.com:9440/mydb?sslmode=" + queryVal + "&channel_binding=" + queryVal2

	got := SummarizeURLDSN(in)

	for _, leak := range []string{username, password, queryVal, queryVal2} {
		if strings.Contains(string(got), leak) {
			t.Fatalf("SummarizeURLDSN leaked sensitive substring %q in %q", leak, got)
		}
	}
}

func TestPhaseError(t *testing.T) {
	t.Parallel()

	cause := errors.New("context deadline exceeded")
	got := PhaseError(
		"CLOUD_DATABASE_URL",
		"cloud-extension",
		DSNSummary("postgres://<redacted>@db.example.com:5432/cloud?keys=tls"),
		"connect",
		cause,
	)
	want := "hatchet-migrate[CLOUD_DATABASE_URL cloud-extension]: failed during connect on " +
		"postgres://<redacted>@db.example.com:5432/cloud?keys=tls: context deadline exceeded"
	if got.Error() != want {
		t.Fatalf("PhaseError mismatch:\n got: %q\nwant: %q", got.Error(), want)
	}
	if !errors.Is(got, cause) {
		t.Fatalf("PhaseError did not wrap original error: %v", got)
	}
}

func TestMissingEnvError(t *testing.T) {
	t.Parallel()

	got := MissingEnvError("DATABASE_URL", "oss")
	want := "hatchet-migrate[DATABASE_URL oss]: failed during read env on <empty DSN>: missing database environment variable"
	if got.Error() != want {
		t.Fatalf("MissingEnvError mismatch:\n got: %q\nwant: %q", got.Error(), want)
	}
	if !errors.Is(got, ErrMissingEnv) {
		t.Fatalf("MissingEnvError did not wrap ErrMissingEnv: %v", got)
	}
}

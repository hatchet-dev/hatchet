package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractClaimsFromJWT(t *testing.T) {
	token := "eyJhbGciOiJFUzI1NiIsICJraWQiOiJRMzNPaGcifQ.eyJhdWQiOiJodHRwczovL2FwcC5kZXYuaGF0Y2hldC10b29scy5jb20iLCAiZXhwIjoxNzE0ODc4NDEyLCAiZ3JwY19icm9hZGNhc3RfYWRkcmVzcyI6IjEyNy4wLjAuMTo3MDcwIiwgImlhdCI6MTcwNzEwMjQxMiwgImlzcyI6Imh0dHBzOi8vYXBwLmRldi5oYXRjaGV0LXRvb2xzLmNvbSIsICJzZXJ2ZXJfdXJsIjoiaHR0cHM6Ly9hcHAuZGV2LmhhdGNoZXQtdG9vbHMuY29tIiwgInN1YiI6IjcwN2QwODU1LTgwYWItNGUxZi1hMTU2LWYxYzQ1NDZjYmY1MiIsICJ0b2tlbl9pZCI6IjI1NzFkODMwLWFmNDgtNDYyZS1hNDFlLTRlZWJkMjUwN2I0NyJ9.abcdefg" // #nosec G101

	claims, err := extractClaimsFromJWT(token)

	assert.Nil(t, err)

	assert.Equal(t, claims["server_url"], "https://app.dev.hatchet-tools.com")
	assert.Equal(t, claims["grpc_broadcast_address"], "127.0.0.1:7070")
}

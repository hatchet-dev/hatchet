//go:build e2e_cli

package cli

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

func TestTokensCreateListRevokeJSON(t *testing.T) {
	h := testharness.New(t)

	name := fmt.Sprintf("e2e-token-%d", time.Now().UnixNano())

	createOut := h.RunJSON("tokens", "create", "--name", name)
	var created struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("failed to unmarshal tokens create output: %v\nOutput: %s", err, createOut)
	}
	if created.Token == "" {
		t.Fatalf("expected non-empty token value, got: %s", createOut)
	}

	listOut := h.RunJSON("tokens", "list")
	var list struct {
		Rows []struct {
			Name     string `json:"name"`
			Metadata struct {
				Id string `json:"id"`
			} `json:"metadata"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(listOut, &list); err != nil {
		t.Fatalf("failed to unmarshal tokens list output: %v\nOutput: %s", err, listOut)
	}

	tokenID := ""
	for _, row := range list.Rows {
		if row.Name == name {
			tokenID = row.Metadata.Id
		}
	}
	if tokenID == "" {
		t.Fatalf("expected to find token %q in list, got: %s", name, listOut)
	}

	revokeOut := h.RunJSON("tokens", "revoke", "--yes", tokenID)
	var revoked struct {
		Revoked bool `json:"revoked"`
	}
	if err := json.Unmarshal(revokeOut, &revoked); err != nil {
		t.Fatalf("failed to unmarshal tokens revoke output: %v\nOutput: %s", err, revokeOut)
	}
	if !revoked.Revoked {
		t.Errorf("expected revoked=true, got: %s", revokeOut)
	}

	afterOut := h.RunJSON("tokens", "list")
	if err := json.Unmarshal(afterOut, &list); err != nil {
		t.Fatalf("failed to unmarshal tokens list output: %v\nOutput: %s", err, afterOut)
	}
	for _, row := range list.Rows {
		if row.Metadata.Id == tokenID {
			t.Errorf("expected token %s to be revoked and absent from list", tokenID)
		}
	}
}

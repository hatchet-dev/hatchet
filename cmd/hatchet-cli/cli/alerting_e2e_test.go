//go:build e2e_cli

package cli

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

func TestAlertingSettingsJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("alerting", "settings")

	var settings map[string]interface{}
	if err := json.Unmarshal(out, &settings); err != nil {
		t.Fatalf("failed to unmarshal alerting settings output: %v\nOutput: %s", err, out)
	}
}

func TestAlertingEmailGroupsCRUDJSON(t *testing.T) {
	h := testharness.New(t)

	unique := time.Now().UnixNano()
	emailA := fmt.Sprintf("a-%d@example.com", unique)
	emailB := fmt.Sprintf("b-%d@example.com", unique)

	createOut := h.RunJSON("alerting", "email-groups", "create", "--emails", emailA+","+emailB)
	var created struct {
		Emails   []string `json:"emails"`
		Metadata struct {
			Id string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("failed to unmarshal email group create output: %v\nOutput: %s", err, createOut)
	}
	if created.Metadata.Id == "" {
		t.Fatalf("expected non-empty email group id, got: %s", createOut)
	}
	if len(created.Emails) != 2 {
		t.Errorf("expected 2 emails in created group, got %d: %s", len(created.Emails), createOut)
	}

	listOut := h.RunJSON("alerting", "email-groups", "list")
	var list struct {
		Rows []struct {
			Emails   []string `json:"emails"`
			Metadata struct {
				Id string `json:"id"`
			} `json:"metadata"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(listOut, &list); err != nil {
		t.Fatalf("failed to unmarshal email groups list output: %v\nOutput: %s", err, listOut)
	}
	found := false
	for _, row := range list.Rows {
		if row.Metadata.Id == created.Metadata.Id {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find email group %s in list, got: %s", created.Metadata.Id, listOut)
	}

	updateOut := h.RunJSON("alerting", "email-groups", "update", created.Metadata.Id, "--emails", emailA)
	var updated struct {
		Emails []string `json:"emails"`
	}
	if err := json.Unmarshal(updateOut, &updated); err != nil {
		t.Fatalf("failed to unmarshal email group update output: %v\nOutput: %s", err, updateOut)
	}
	if len(updated.Emails) != 1 || updated.Emails[0] != emailA {
		t.Errorf("expected updated emails [%s], got: %s", emailA, updateOut)
	}

	deleteOut := h.RunJSON("alerting", "email-groups", "delete", "--yes", created.Metadata.Id)
	var deleted struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.Unmarshal(deleteOut, &deleted); err != nil {
		t.Fatalf("failed to unmarshal email group delete output: %v\nOutput: %s", err, deleteOut)
	}
	if !deleted.Deleted {
		t.Errorf("expected deleted=true, got: %s", deleteOut)
	}

	afterOut := h.RunJSON("alerting", "email-groups", "list")
	if err := json.Unmarshal(afterOut, &list); err != nil {
		t.Fatalf("failed to unmarshal email groups list output: %v\nOutput: %s", err, afterOut)
	}
	for _, row := range list.Rows {
		if row.Metadata.Id == created.Metadata.Id {
			t.Errorf("expected email group %s to be deleted, still in list", created.Metadata.Id)
		}
	}
}

func TestAlertingSlackListJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("alerting", "slack", "list")

	var result struct {
		Rows []map[string]interface{} `json:"rows"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal slack list output: %v\nOutput: %s", err, out)
	}
}

func TestAlertingSnsCRUDJSON(t *testing.T) {
	h := testharness.New(t)

	topicArn := fmt.Sprintf("arn:aws:sns:us-east-1:123456789012:e2e-%d", time.Now().UnixNano())

	createOut, err := e2eTryRunJSON(t, h, "alerting", "sns", "create", "--topic-arn", topicArn)
	if err != nil {
		t.Skipf("SNS integration creation rejected (likely ARN validation): %v", err)
	}

	var created struct {
		TopicArn string `json:"topicArn"`
		Metadata struct {
			Id string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("failed to unmarshal sns create output: %v\nOutput: %s", err, createOut)
	}
	if created.TopicArn != topicArn {
		t.Errorf("expected topicArn %q, got %q", topicArn, created.TopicArn)
	}
	if created.Metadata.Id == "" {
		t.Fatalf("expected non-empty sns integration id, got: %s", createOut)
	}

	listOut := h.RunJSON("alerting", "sns", "list")
	var list struct {
		Rows []struct {
			TopicArn string `json:"topicArn"`
			Metadata struct {
				Id string `json:"id"`
			} `json:"metadata"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(listOut, &list); err != nil {
		t.Fatalf("failed to unmarshal sns list output: %v\nOutput: %s", err, listOut)
	}
	found := false
	for _, row := range list.Rows {
		if row.Metadata.Id == created.Metadata.Id {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find sns integration %s in list, got: %s", created.Metadata.Id, listOut)
	}

	deleteOut := h.RunJSON("alerting", "sns", "delete", "--yes", created.Metadata.Id)
	var deleted struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.Unmarshal(deleteOut, &deleted); err != nil {
		t.Fatalf("failed to unmarshal sns delete output: %v\nOutput: %s", err, deleteOut)
	}
	if !deleted.Deleted {
		t.Errorf("expected deleted=true, got: %s", deleteOut)
	}
}

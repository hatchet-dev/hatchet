package postmark

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/integrations/email"
)

func TestSendMagicLinkEmailUsesTemplate(t *testing.T) {
	const tokenURL = "https://cloud.example/auth/magic-link#one-time-token"

	var gotPath string
	var got struct {
		To            string                   `json:"To"`
		TemplateAlias string                   `json:"TemplateAlias"`
		TemplateModel email.MagicLinkEmailData `json:"TemplateModel"`
	}
	var gotToken string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Postmark-Server-Token")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("decode body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewPostmarkClient("server-token", "support@example.com", "Hatchet", "support@example.com")
	client.apiURL = server.URL

	err := client.SendMagicLinkEmail(t.Context(), "user@example.com", email.MagicLinkEmailData{
		ActionURL: tokenURL,
		ExpiresIn: "15 minutes",
	})
	if err != nil {
		t.Fatalf("SendMagicLinkEmail: %v", err)
	}

	if gotPath != postmarkEmailPath {
		t.Fatalf("path = %s, want template email endpoint", gotPath)
	}
	if gotToken != "server-token" {
		t.Fatalf("server token header = %q", gotToken)
	}
	if got.To != "user@example.com" {
		t.Fatalf("to = %q", got.To)
	}
	if got.TemplateAlias != email.MagicLinkTemplate {
		t.Fatalf("template alias = %q", got.TemplateAlias)
	}
	if got.TemplateModel.ActionURL != tokenURL || got.TemplateModel.ExpiresIn != "15 minutes" {
		t.Fatalf("template model = %+v", got.TemplateModel)
	}
}

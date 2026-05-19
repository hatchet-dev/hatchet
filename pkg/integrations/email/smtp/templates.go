package smtp

import (
	"embed"
	"fmt"
	html "html/template"
	text "text/template"

	"github.com/hatchet-dev/hatchet/pkg/integrations/email"
)

//go:embed templates/*.html
var templateFS embed.FS

var templateRegistry = make(map[string]struct {
	bodyTmpl    *html.Template
	subjectTmpl *text.Template
})

const (
	subjectTemplateAlias   = "subject"
	inviteSubjectTemplate  = `{{.InviteSenderName}} invited you to join {{.TenantName}} on Hatchet`
	defaultSubjectTemplate = `[{{.TenantName}}] {{.Subject}}`
)

func init() {
	templates := map[string]struct {
		fileName string
		subject  string
	}{
		email.UserInviteTemplate:         {"templates/user_invite.html", inviteSubjectTemplate},
		email.OrganizationInviteTemplate: {"templates/organization_invite.html", inviteSubjectTemplate},
		email.TokenAlertExpiringTemplate: {"templates/expiring_token.html", defaultSubjectTemplate},
		email.ResourceLimitAlertTemplate: {"templates/resource_limit_alert.html", defaultSubjectTemplate},
		email.WorkflowRunsFailedTemplate: {"templates/workflow_runs_failed.html", defaultSubjectTemplate},
	}

	for alias, tmpl := range templates {
		// We need to ensure that the layout.html is parsed before the HTML template
		bodyTmpl := html.Must(
			html.ParseFS(templateFS, "templates/layout.html", tmpl.fileName),
		)

		subjectTmpl := text.Must(text.New("email-subject").Parse(tmpl.subject))

		templateRegistry[alias] = struct {
			bodyTmpl    *html.Template
			subjectTmpl *text.Template
		}{
			bodyTmpl:    bodyTmpl,
			subjectTmpl: subjectTmpl,
		}

	}
}

func getEmailTemplate(alias string) (*html.Template, error) {
	tmpl, ok := templateRegistry[alias]
	if !ok {
		return nil, fmt.Errorf("template %s does not exist", alias)
	}
	return tmpl.bodyTmpl, nil
}

func getSubjectTemplate(alias string) (*text.Template, error) {
	tmpl, ok := templateRegistry[alias]
	if !ok {
		return nil, fmt.Errorf("template %s does not exist", alias)
	}
	return tmpl.subjectTmpl, nil
}

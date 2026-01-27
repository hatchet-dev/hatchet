package smtp

import (
	"embed"
	"fmt"
	"html/template"

	"github.com/hatchet-dev/hatchet/pkg/integrations/email"
)

//go:embed templates/*.html
var templateFS embed.FS

var templateRegistry = make(map[string]*template.Template)

func init() {
	templates := map[string]string{
		email.TokenAlertExpiringTemplate: "templates/expiring_token.html",
		email.UserInviteTemplate:         "templates/user_invite.html",
		email.OrganizationInviteTemplate: "templates/organization_invite.html",
		email.ResourceLimitAlertTemplate: "templates/resource_limit_alert.html",
		email.WorkflowRunsFailedTemplate: "templates/workflow_runs_failed.html",
	}

	for alias, fileName := range templates {
		// We need to ensure that the layout.html is parsed before the HTML template
		templateRegistry[alias] = template.Must(
			template.ParseFS(templateFS, "templates/layout.html", fileName),
		)
	}
}

func getTemplate(alias string) (*template.Template, error) {
	tmpl, ok := templateRegistry[alias]
	if !ok {
		return nil, fmt.Errorf("template %s does not exist", alias)
	}
	return tmpl, nil
}

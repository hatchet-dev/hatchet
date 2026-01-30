package smtp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/wneessen/go-mail"

	"github.com/hatchet-dev/hatchet/pkg/integrations/email"
)

const defaultSMTPClientPort = 587

var (
	_ email.EmailService = &SMTPService{}

	errNoSubject = errors.New("subject field of email cannot be empty")
)

type SMTPService struct {
	fromEmail    string
	fromName     string
	supportEmail string
	client       *mail.Client
}

// NewSMTPService creates a new service which sends emails
func NewSMTPService(serverAddr, serverUser, serverKey, fromEmail, fromName, supportEmail string) (*SMTPService, error) {
	port := defaultSMTPClientPort
	if host, portStr, err := net.SplitHostPort(serverAddr); err == nil {
		// if we can split the host by port, then override default
		serverAddr = host
		port, _ = strconv.Atoi(portStr)
	}

	authType := mail.SMTPAuthPlain
	if serverUser == "" || serverKey == "" {
		// No username or password is provided, we assume auth is disabled.
		authType = mail.SMTPAuthNoAuth
	}

	client, err := mail.NewClient(serverAddr,
		mail.WithPort(port),
		mail.WithTLSPortPolicy(mail.TLSOpportunistic),
		mail.WithSMTPAuth(authType),
		mail.WithUsername(serverUser),
		mail.WithPassword(serverKey),
	)
	if err != nil {
		return nil, err
	}

	return &SMTPService{
		client:       client,
		fromEmail:    fromEmail,
		fromName:     fromName,
		supportEmail: supportEmail,
	}, nil
}

func (s *SMTPService) IsValid() bool {
	return true
}

func (s *SMTPService) SendTenantInviteEmail(ctx context.Context, to string, data email.TenantInviteEmailData) error {
	return s.SendTemplateEmail(ctx, to, email.UserInviteTemplate, data, false)
}

func (s *SMTPService) SendWorkflowRunFailedAlerts(ctx context.Context, emails []string, data email.WorkflowRunsFailedEmailData) error {
	return s.SendTemplateEmailBCC(ctx, strings.Join(emails, ","), email.WorkflowRunsFailedTemplate, data, false)
}

func (s *SMTPService) SendExpiringTokenEmail(ctx context.Context, emails []string, data email.ExpiringTokenEmailData) error {
	return s.SendTemplateEmailBCC(ctx, strings.Join(emails, ","), email.TokenAlertExpiringTemplate, data, false)
}

func (s *SMTPService) SendTenantResourceLimitAlert(ctx context.Context, emails []string, data email.ResourceLimitAlertData) error {
	return s.SendTemplateEmailBCC(ctx, strings.Join(emails, ","), email.ResourceLimitAlertTemplate, data, true)
}

func (s *SMTPService) SendTemplateEmail(ctx context.Context, to, templateAlias string, templateModelData any, bccSupport bool) error {
	var bcc string

	if bccSupport {
		bcc = s.supportEmail
	}

	return s.sendRequest(ctx, &email.SendEmailFromTemplateRequest{
		From:          fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:            to,
		Bcc:           bcc,
		TemplateAlias: templateAlias,
		TemplateModel: templateModelData,
	})
}

func (s *SMTPService) SendTemplateEmailBCC(ctx context.Context, bcc, templateAlias string, templateModelData any, bccSupport bool) error {
	if bccSupport {
		bcc = fmt.Sprintf("%s,%s", bcc, s.supportEmail)
	}

	return s.sendRequest(ctx, &email.SendEmailFromTemplateRequest{
		From:          fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:            "",
		Bcc:           bcc,
		TemplateAlias: templateAlias,
		TemplateModel: templateModelData,
	})
}

func (s *SMTPService) sendRequest(ctx context.Context, req *email.SendEmailFromTemplateRequest) error {
	msg := mail.NewMsg()

	if req.From != "" {
		if err := msg.From(req.From); err != nil {
			return err
		}
	}

	if req.Bcc != "" {
		if err := msg.BccFromString(req.Bcc); err != nil {
			return err
		}
	}

	if req.To != "" {
		if err := msg.ToFromString(req.To); err != nil {
			return err
		}
	}

	subjectTmpl, err := getSubjectTemplate(req.TemplateAlias)
	if err != nil {
		return err
	}

	var subject bytes.Buffer
	err = subjectTmpl.Execute(&subject, req.TemplateModel)
	if err != nil {
		return err
	}

	subj := subject.String()
	if subj == "" {
		return errNoSubject
	}
	msg.Subject(subj)

	tmpl, err := getEmailTemplate(req.TemplateAlias)
	if err != nil {
		return err
	}

	if err := msg.SetBodyHTMLTemplate(tmpl, req.TemplateModel); err != nil {
		return err
	}

	return s.client.DialAndSendWithContext(ctx, msg)
}

//go:build !e2e && !load && !rampup && !integration

package smtp

import (
	"io"
	"log"
	"net"
	"net/mail"
	"strconv"
	"sync"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

func StartMockSMTPServer() (port int, capture *SMTPCapture, cancelFunc func(), err error) {
	// TODO(gregfurman): This mock SMTP server approach is _good enough_ to test our SMTP client against
	// and perform introspection on request/responses. However, it's probably a better idea to use an external
	// container or service (i.e https://github.com/sj26/mailcatcher) since this approach is not easily maintainable.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, nil, err
	}

	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return 0, nil, nil, err
	}

	port, err = strconv.Atoi(portStr)
	if err != nil {
		return 0, nil, nil, err
	}

	capture = &SMTPCapture{}
	s := smtp.NewServer(&mockBackend{capture: capture})
	s.Addr = listener.Addr().String()
	s.Domain = "localhost"
	s.AllowInsecureAuth = true

	go func() {
		if err := s.Serve(listener); err != nil && err != net.ErrClosed {
			log.Printf("SMTP mock server error: %v", err)
		}
	}()

	return port, capture, func() { s.Close() }, nil
}

// ----------------------------------------------------------------------

var (
	_ smtp.Backend = &mockBackend{}
	_ smtp.Session = &mockSession{}
)

// SMTPCapture captures SMTP request and response data that can be used to
// perform assertions.
type SMTPCapture struct {
	mu        sync.Mutex
	Usernames []string
	Passwords []string
	Froms     []string
	Rcpts     []string
	Messages  []*mail.Message
}

type mockBackend struct {
	capture *SMTPCapture
}

func (b *mockBackend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &mockSession{capture: b.capture}, nil
}

type mockSession struct {
	capture *SMTPCapture
}

func (s *mockSession) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

func (s *mockSession) Auth(mech string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(identity, username, password string) error {
		s.capture.mu.Lock()
		defer s.capture.mu.Unlock()
		s.capture.Usernames = append(s.capture.Usernames, username)
		s.capture.Passwords = append(s.capture.Passwords, password)
		return nil
	}), nil
}

func (s *mockSession) Mail(from string, _ *smtp.MailOptions) error {
	s.capture.mu.Lock()
	defer s.capture.mu.Unlock()
	s.capture.Froms = append(s.capture.Froms, from)
	return nil
}

func (s *mockSession) Rcpt(to string, _ *smtp.RcptOptions) error {
	s.capture.mu.Lock()
	defer s.capture.mu.Unlock()
	s.capture.Rcpts = append(s.capture.Rcpts, to)
	return nil
}

func (s *mockSession) Data(r io.Reader) error {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return err
	}

	s.capture.mu.Lock()
	defer s.capture.mu.Unlock()
	s.capture.Messages = append(s.capture.Messages, msg)
	return nil
}

func (*mockSession) Reset() {}

func (*mockSession) Logout() error {
	return nil
}

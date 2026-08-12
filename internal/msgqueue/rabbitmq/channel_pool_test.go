package rabbitmq

import "testing"

func TestRedactURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "username and password",
			in:   "amqp://user:s3cret@rabbitmq-blue:5672/",
			want: "amqp://user:xxxxx@rabbitmq-blue:5672/",
		},
		{
			name: "username only",
			in:   "amqp://user@rabbitmq:5672/vhost",
			want: "amqp://user@rabbitmq:5672/vhost",
		},
		{
			name: "no credentials",
			in:   "amqp://rabbitmq:5672/",
			want: "amqp://rabbitmq:5672/",
		},
		{
			name: "unparseable",
			in:   "amqp://user:pass@rabbit:5672/\x7f%zz",
			want: "<unparseable url>",
		},
		{
			// A scheme-less string parses without a host, and Redacted would
			// return the credentials verbatim, so it must be masked entirely.
			name: "scheme-less with credentials",
			in:   "user:pass@rabbit:5672/vhost",
			want: "<unparseable url>",
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactURL(tt.in)

			if got != tt.want {
				t.Errorf("redactURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

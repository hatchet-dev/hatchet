package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPubSubSettingsInheritance checks the config backwards-compatibility
// invariant: a deployment configured only with today's durable message queue
// variables (including the legacy SERVER_TASKQUEUE_* aliases) resolves to a
// fully configured pub/sub path with zero new variables.
func TestPubSubSettingsInheritance(t *testing.T) {
	cases := []struct {
		name       string
		env        map[string]string
		wantKind   string
		wantURL    string
		wantMaxPub int32
		wantMaxSub int32
	}{
		{
			name: "modern rabbit env only",
			env: map[string]string{
				"SERVER_MSGQUEUE_KIND":         "rabbitmq",
				"SERVER_MSGQUEUE_RABBITMQ_URL": "amqp://user:password@rabbit:5672/",
			},
			wantKind:   "rabbitmq",
			wantURL:    "amqp://user:password@rabbit:5672/",
			wantMaxPub: 10,
			wantMaxSub: 20,
		},
		{
			name: "legacy taskqueue aliases only",
			env: map[string]string{
				"SERVER_TASKQUEUE_KIND":         "rabbitmq",
				"SERVER_TASKQUEUE_RABBITMQ_URL": "amqp://legacy:password@rabbit:5672/",
			},
			wantKind:   "rabbitmq",
			wantURL:    "amqp://legacy:password@rabbit:5672/",
			wantMaxPub: 10,
			wantMaxSub: 20,
		},
		{
			name: "postgres durable only",
			env: map[string]string{
				"SERVER_MSGQUEUE_KIND": "postgres",
			},
			wantKind:   "postgres",
			wantURL:    "",
			wantMaxPub: 10,
			wantMaxSub: 20,
		},
		{
			name: "explicit pubsub overrides",
			env: map[string]string{
				"SERVER_MSGQUEUE_KIND":                          "postgres",
				"SERVER_MSGQUEUE_PUBSUB_KIND":                   "rabbitmq",
				"SERVER_MSGQUEUE_PUBSUB_RABBITMQ_URL":           "amqp://pubsub:password@rabbit:5672/",
				"SERVER_MSGQUEUE_PUBSUB_RABBITMQ_MAX_PUB_CHANS": "3",
				"SERVER_MSGQUEUE_PUBSUB_RABBITMQ_MAX_SUB_CHANS": "7",
			},
			wantKind:   "rabbitmq",
			wantURL:    "amqp://pubsub:password@rabbit:5672/",
			wantMaxPub: 3,
			wantMaxSub: 7,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			cf, err := LoadServerConfigFile()
			require.NoError(t, err)

			kind, url := resolvePubSubKindAndURL(cf)

			assert.Equal(t, tc.wantKind, kind)
			assert.Equal(t, tc.wantURL, url)
			assert.Equal(t, tc.wantMaxPub, cf.MessageQueue.PubSub.RabbitMQ.MaxPubChans)
			assert.Equal(t, tc.wantMaxSub, cf.MessageQueue.PubSub.RabbitMQ.MaxSubChans)
		})
	}
}

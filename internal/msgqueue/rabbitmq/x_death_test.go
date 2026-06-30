package rabbitmq

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestXDeathCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		xDeath []interface{}
		want   int64
	}{
		{
			name: "rabbitmq int32 count",
			xDeath: []interface{}{
				amqp.Table{"count": int32(3)},
			},
			want: 3,
		},
		{
			name: "int64 count",
			xDeath: []interface{}{
				amqp.Table{"count": int64(7)},
			},
			want: 7,
		},
		{
			name:   "empty x-death",
			xDeath: []interface{}{},
			want:   0,
		},
		{
			name: "missing count",
			xDeath: []interface{}{
				amqp.Table{"reason": "rejected"},
			},
			want: 0,
		},
		{
			name: "unexpected entry type",
			xDeath: []interface{}{
				"not-a-table",
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, xDeathCount(tt.xDeath))
		})
	}
}

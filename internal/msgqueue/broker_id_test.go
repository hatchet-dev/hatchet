package msgqueue

import "testing"

type brokerIdentifyingMessageQueue struct {
	mockMessageQueue
	id string
}

func (m *brokerIdentifyingMessageQueue) BrokerID() string { return m.id }

func TestBrokerID(t *testing.T) {
	cases := []struct {
		name string
		mq   MessageQueue
		want string
	}{
		{name: "nil", mq: nil, want: ""},
		{name: "does_not_identify_broker", mq: &mockMessageQueue{}, want: ""},
		{name: "identifies_broker", mq: &brokerIdentifyingMessageQueue{id: "broker-a"}, want: "broker-a"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BrokerID(tc.mq); got != tc.want {
				t.Fatalf("BrokerID() = %q, want %q", got, tc.want)
			}
		})
	}
}

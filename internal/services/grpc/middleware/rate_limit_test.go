package middleware

import "testing"

func TestMatchServiceName(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{name: "/Dispatcher/Listen", want: "dispatcher"},
		{name: "/v1.V1Dispatcher/DurableTask", want: "dispatcher"},
		{name: "/v1.OperatorService/Listen", want: "dispatcher"},
		{name: "/v1.OperatorService/SendStepActionEvent", want: "dispatcher"},
		{name: "/EventsService/Push", want: "events"},
		{name: "/WorkflowService/PutWorkflow", want: "workflow"},
		{name: "/v1.AdminService/PutWorkflow", want: "admin"},
		{name: "/opentelemetry.proto.collector.trace.v1.TraceService/Export", want: "otelcol"},
		{name: "/something.Else/Method", want: "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchServiceName(tc.name); got != tc.want {
				t.Fatalf("matchServiceName(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

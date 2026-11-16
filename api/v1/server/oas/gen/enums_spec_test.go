package gen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/enums"
)

func values[T ~string](all []T) []string {
	out := make([]string, len(all))
	for i, v := range all {
		out[i] = string(v)
	}
	return out
}

// x-go-type drops the spec's enum list from the Go output, so this is the only check that the API and database agree
var enumValues = map[string][]string{
	"LimitResource":                  values(enums.AllLimitResourceValues()),
	"TenantMajorEngineVersion":       values(enums.AllTenantMajorEngineVersionValues()),
	"TenantMemberRole":               values(enums.AllTenantMemberRoleValues()),
	"V1ConcurrencyStrategy":          values(enums.AllV1ConcurrencyStrategyValues()),
	"V1DurableEventLogKind":          values(enums.AllV1DurableEventLogKindValues()),
	"V1EventTypeOlap":                values(enums.AllV1EventTypeOlapValues()),
	"V1IncomingWebhookAuthType":      values(enums.AllV1IncomingWebhookAuthTypeValues()),
	"V1IncomingWebhookHmacAlgorithm": values(enums.AllV1IncomingWebhookHmacAlgorithmValues()),
	"V1IncomingWebhookSourceName":    values(enums.AllV1IncomingWebhookSourceNameValues()),
	"V1RunKind":                      values(enums.AllV1RunKindValues()),
	"V1ServerlessEndpointEventKind":  values(enums.AllV1ServerlessEndpointEventKindValues()),
	"V1ServerlessEndpointKind":       values(enums.AllV1ServerlessEndpointKindValues()),
	"V1TaskStatus":                   values(enums.AllV1TaskStatusValues()),
}

func TestSharedEnumsMatchSpec(t *testing.T) {
	spec, err := gen.GetSwagger()
	if err != nil {
		t.Fatal(err)
	}

	mapped := map[string]bool{}

	for name, ref := range spec.Components.Schemas {
		goType, _ := ref.Value.Extensions["x-go-type"].(string)
		typeName, ok := strings.CutPrefix(goType, "enums.")
		if !ok {
			continue
		}
		mapped[typeName] = true

		want, ok := enumValues[typeName]
		if !ok {
			t.Errorf("schema %s maps to enums.%s, which has no entry in enumValues", name, typeName)
			continue
		}

		got := make([]string, 0, len(ref.Value.Enum))
		for _, v := range ref.Value.Enum {
			s, ok := v.(string)
			if !ok {
				t.Errorf("schema %s has non-string enum value %v", name, v)
				continue
			}
			got = append(got, s)
		}

		slices.Sort(got)
		want = slices.Sorted(slices.Values(want))

		if !slices.Equal(got, want) {
			t.Errorf("schema %s and enums.%s disagree:\n  spec:     %v\n  database: %v", name, typeName, got, want)
		}
	}

	for typeName := range enumValues {
		if !mapped[typeName] {
			t.Errorf("enumValues has enums.%s but no schema maps to it via x-go-type", typeName)
		}
	}
}

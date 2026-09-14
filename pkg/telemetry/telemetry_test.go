package telemetry

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestK8sResourceAttributesCloudRegion(t *testing.T) {
	t.Setenv("K8S_POD_NAME", "engine-abc")
	t.Setenv("K8S_POD_NAMESPACE", "sixfold")
	t.Setenv("K8S_CLOUD_REGION", "us-east-1")
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_DEFAULT_REGION", "ap-southeast-2")

	got := attrMap(k8sResourceAttributes())

	if got["k8s.pod.name"] != "engine-abc" {
		t.Fatalf("k8s.pod.name = %q", got["k8s.pod.name"])
	}

	if got["k8s.namespace.name"] != "sixfold" {
		t.Fatalf("k8s.namespace.name = %q", got["k8s.namespace.name"])
	}

	if got["cloud.region"] != "us-east-1" {
		t.Fatalf("cloud.region = %q, want K8S_CLOUD_REGION over AWS_REGION", got["cloud.region"])
	}
}

func TestK8sResourceAttributesCloudRegionFallsBackToAWSRegion(t *testing.T) {
	t.Setenv("K8S_CLOUD_REGION", "")
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_DEFAULT_REGION", "ap-southeast-2")

	got := attrMap(k8sResourceAttributes())

	if got["cloud.region"] != "eu-west-1" {
		t.Fatalf("cloud.region = %q, want AWS_REGION", got["cloud.region"])
	}
}

func attrMap(attrs []attribute.KeyValue) map[string]string {
	out := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		out[string(attr.Key)] = attr.Value.AsString()
	}

	return out
}

func TestBuildHeadersOnlyAuth(t *testing.T) {
	got := buildHeaders(&TracerOpts{CollectorAuth: "token"})

	if got["Authorization"] != "token" {
		t.Fatalf("Authorization = %q, want %q", got["Authorization"], "token")
	}

	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
}

func TestBuildHeadersOnlyHeaders(t *testing.T) {
	got := buildHeaders(&TracerOpts{
		CollectorHeaders: map[string]string{"X-API-Key": "k"},
	})

	if got["X-API-Key"] != "k" {
		t.Fatalf("X-API-Key = %q, want %q", got["X-API-Key"], "k")
	}

	if got["Authorization"] != "" {
		t.Fatalf("Authorization = %q, want %q (CollectorAuth empty, preserves prior behavior)", got["Authorization"], "")
	}

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func TestBuildHeadersExplicitAuthWins(t *testing.T) {
	got := buildHeaders(&TracerOpts{
		CollectorAuth:    "old",
		CollectorHeaders: map[string]string{"Authorization": "Bearer xyz"},
	})

	if got["Authorization"] != "Bearer xyz" {
		t.Fatalf("Authorization = %q, want explicit headers value %q", got["Authorization"], "Bearer xyz")
	}
}

func TestBuildHeadersNonConflictMerge(t *testing.T) {
	got := buildHeaders(&TracerOpts{
		CollectorAuth:    "tok",
		CollectorHeaders: map[string]string{"X-Tenant": "t1", "X-Trace": "yes"},
	})

	if got["Authorization"] != "tok" {
		t.Fatalf("Authorization = %q, want %q", got["Authorization"], "tok")
	}
	if got["X-Tenant"] != "t1" {
		t.Fatalf("X-Tenant = %q, want %q", got["X-Tenant"], "t1")
	}
	if got["X-Trace"] != "yes" {
		t.Fatalf("X-Trace = %q, want %q", got["X-Trace"], "yes")
	}
}

func TestBuildHeadersNeither(t *testing.T) {
	got := buildHeaders(&TracerOpts{})

	if got["Authorization"] != "" {
		t.Fatalf("Authorization = %q, want %q (CollectorAuth empty, preserves prior behavior)", got["Authorization"], "")
	}

	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
}

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

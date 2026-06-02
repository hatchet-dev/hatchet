package main

import (
	"strings"
	"testing"
)

func TestBuildOperationMetadataMap_hasTenantPathFromOpenAPIPath(t *testing.T) {
	t.Parallel()

	spec := OpenAPISpec{
		Paths: map[string]PathItem{
			"/api/v1/tenants/{tenant}/workflows": {
				Get: &Operation{XResources: []string{"tenant"}},
			},
			"/api/v1/stable/tasks/{task}": {
				Get: &Operation{XResources: []string{"tenant", "task"}},
			},
		},
	}

	m := buildOperationMetadataMap(spec)

	tenantMeta := m["GET /api/v1/tenants/{tenant}/workflows"]
	if !tenantMeta.hasTenantPath {
		t.Fatal("expected hasTenantPath for {tenant} path")
	}
	if len(tenantMeta.resources) != 1 || tenantMeta.resources[0] != "tenant" {
		t.Fatalf("unexpected resources: %#v", tenantMeta.resources)
	}

	taskMeta := m["GET /api/v1/stable/tasks/{task}"]
	if taskMeta.hasTenantPath {
		t.Fatal("expected hasTenantPath false for task path without {tenant}")
	}
	if len(taskMeta.resources) != 2 {
		t.Fatalf("unexpected resources: %#v", taskMeta.resources)
	}
}

func TestTransformMethod_tenantPathInjectsXTenantIdBeforeParams(t *testing.T) {
	t.Parallel()

	input := []string{
		"  tenantGet = (tenant: string, params: RequestParams = {}) =>",
		"    this.request<Tenant, APIErrors>({",
		"      path: " + "`/api/v1/tenants/${tenant}`" + ",",
		"      method: \"GET\",",
		"      secure: true,",
		"      format: \"json\",",
		"      ...params,",
		"    });",
	}

	got := transformMethod(input, operationMetadata{
		resources:     []string{"tenant"},
		hasTenantPath: true,
	})

	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "xTenantId: tenant,") {
		t.Fatalf("missing xTenantId injection:\n%s", joined)
	}
	if !strings.Contains(joined, "xResources: [\"tenant\"],") {
		t.Fatalf("missing xResources injection:\n%s", joined)
	}

	tenantIdx := strings.Index(joined, "xTenantId: tenant,")
	paramsIdx := strings.Index(joined, "...params,")
	resourcesIdx := strings.Index(joined, "xResources:")
	if tenantIdx == -1 || paramsIdx == -1 || resourcesIdx == -1 {
		t.Fatalf("missing expected lines:\n%s", joined)
	}
	if !(tenantIdx < paramsIdx && paramsIdx < resourcesIdx) {
		t.Fatalf("expected xTenantId before ...params before xResources:\n%s", joined)
	}
}

func TestTransformMethod_nonTenantPathSkipsXTenantId(t *testing.T) {
	t.Parallel()

	input := []string{
		"  v1TaskGet = (task: string, params: RequestParams = {}) =>",
		"    this.request<V1TaskSummary, APIErrors>({",
		"      path: " + "`/api/v1/stable/tasks/${task}`" + ",",
		"      method: \"GET\",",
		"      secure: true,",
		"      format: \"json\",",
		"      ...params,",
		"    });",
	}

	got := transformMethod(input, operationMetadata{
		resources:     []string{"tenant", "task"},
		hasTenantPath: false,
	})

	joined := strings.Join(got, "\n")
	if strings.Contains(joined, "xTenantId:") {
		t.Fatalf("unexpected xTenantId for non-tenant path:\n%s", joined)
	}
}

func TestTransformMethod_tenantResourceWithoutTenantPathSkipsXTenantId(t *testing.T) {
	t.Parallel()

	input := []string{
		"  v1WorkflowRunGet = (v1WorkflowRun: string, params: RequestParams = {}) =>",
		"    this.request<V1WorkflowRunDetails, APIErrors>({",
		"      path: " + "`/api/v1/stable/workflow-runs/${v1WorkflowRun}`" + ",",
		"      method: \"GET\",",
		"      secure: true,",
		"      format: \"json\",",
		"      ...params,",
		"    });",
	}

	got := transformMethod(input, operationMetadata{
		resources:     []string{"tenant", "v1-workflow-run"},
		hasTenantPath: false,
	})

	if strings.Contains(strings.Join(got, "\n"), "xTenantId:") {
		t.Fatal("expected no xTenantId when OpenAPI path lacks {tenant}")
	}
}

func TestTransformApiTs_idempotentWhenAlreadyPatched(t *testing.T) {
	t.Parallel()

	content := "  /**\n" +
		"   * @request GET:/api/v1/tenants/{tenant}\n" +
		"   */\n" +
		"  tenantGet = Object.assign((tenant: string, params: RequestParams = {}) =>\n" +
		"    this.request<Tenant, APIErrors>({\n" +
		"      path: `/api/v1/tenants/${tenant}`,\n" +
		"      method: \"GET\",\n" +
		"      ...params,\n" +
		"      xResources: [\"tenant\"],\n" +
		"    }), { resources: new Set<string>([\"tenant\"]) });\n"

	metadataMap := map[string]operationMetadata{
		"GET /api/v1/tenants/{tenant}": {
			resources:     []string{"tenant"},
			hasTenantPath: true,
		},
	}

	result, total, _, withTenantPath := transformApiTs(content, metadataMap)
	if total != 0 {
		t.Fatalf("expected no transforms on already-patched method, got %d", total)
	}
	if withTenantPath != 0 {
		t.Fatalf("expected withTenantPath 0, got %d", withTenantPath)
	}
	if result != content {
		t.Fatal("expected content unchanged for idempotent run")
	}
}

func TestTransformApiTs_patchesUnpatchedTenantMethod(t *testing.T) {
	t.Parallel()

	content := "  /**\n" +
		"   * @request GET:/api/v1/tenants/{tenant}/workflows\n" +
		"   */\n" +
		"  workflowList = (\n" +
		"    tenant: string,\n" +
		"    params: RequestParams = {},\n" +
		"  ) =>\n" +
		"    this.request<WorkflowList, APIErrors>({\n" +
		"      path: `/api/v1/tenants/${tenant}/workflows`,\n" +
		"      method: \"GET\",\n" +
		"      secure: true,\n" +
		"      format: \"json\",\n" +
		"      ...params,\n" +
		"    });\n"

	metadataMap := map[string]operationMetadata{
		"GET /api/v1/tenants/{tenant}/workflows": {
			resources:     []string{"tenant"},
			hasTenantPath: true,
		},
	}

	result, total, _, withTenantPath := transformApiTs(content, metadataMap)
	if total != 1 {
		t.Fatalf("expected 1 transform, got %d", total)
	}
	if withTenantPath != 1 {
		t.Fatalf("expected withTenantPath 1, got %d", withTenantPath)
	}
	if !strings.Contains(result, "xTenantId: tenant,") {
		t.Fatalf("expected xTenantId in result:\n%s", result)
	}
	if !strings.Contains(result, "Object.assign(") {
		t.Fatal("expected Object.assign wrapper")
	}
}

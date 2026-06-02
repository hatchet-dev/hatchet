// patch-api-resources patches the generated Api.ts file to attach x-resources
// metadata from the OpenAPI spec to each API method, enabling callers to check
// resource scope at runtime:
//
//	api.v1TaskGet.resources.has("tenant") // true
//
// For operations whose path contains {tenant}, it also injects xTenantId: tenant
// so the exchange-token interceptor can resolve the tenant without URL parsing.
//
// Run from the repo root after swagger-typescript-api generation:
//
//	go run ./hack/oas/patch-api-resources.go
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	specPath = "bin/oas/openapi.yaml"
	apiPath  = "frontend/app/src/lib/api/generated/Api.ts"
)

// OpenAPI types — only the fields we need.

type OpenAPISpec struct {
	Paths map[string]PathItem `yaml:"paths"`
}

type PathItem struct {
	Get    *Operation `yaml:"get"`
	Post   *Operation `yaml:"post"`
	Put    *Operation `yaml:"put"`
	Patch  *Operation `yaml:"patch"`
	Delete *Operation `yaml:"delete"`
}

type Operation struct {
	XResources []string `yaml:"x-resources"`
}

type operationMetadata struct {
	resources     []string
	hasTenantPath bool
}

func main() {
	// Load and parse OpenAPI spec.
	specData, err := os.ReadFile(specPath)
	if err != nil {
		log.Fatalf("reading %s: %v", specPath, err)
	}

	var spec OpenAPISpec
	if err := yaml.Unmarshal(specData, &spec); err != nil { //nolint:govet
		log.Fatalf("parsing %s: %v", specPath, err)
	}

	metadataMap := buildOperationMetadataMap(spec)

	apiData, err := os.ReadFile(apiPath)
	if err != nil {
		log.Fatalf("reading %s: %v", apiPath, err)
	}

	result, total, withResources, withTenantPath := transformApiTs(string(apiData), metadataMap)

	if err := os.WriteFile(apiPath, []byte(result), 0644); err != nil { // nolint:gosec
		log.Fatalf("writing %s: %v", apiPath, err)
	}

	fmt.Printf(
		"patched %s: %d methods transformed (%d with x-resources, %d with xTenantId)\n",
		apiPath, total, withResources, withTenantPath,
	)
}

type resource struct {
	op     *Operation
	method string
}

func buildOperationMetadataMap(spec OpenAPISpec) map[string]operationMetadata {
	m := make(map[string]operationMetadata)
	for path, item := range spec.Paths {
		hasTenantPath := strings.Contains(path, "{tenant}")
		for _, entry := range []resource{
			{op: item.Get, method: "GET"},
			{op: item.Post, method: "POST"},
			{op: item.Put, method: "PUT"},
			{op: item.Patch, method: "PATCH"},
			{op: item.Delete, method: "DELETE"},
		} {
			if entry.op != nil {
				m[entry.method+" "+path] = operationMetadata{
					resources:     entry.op.XResources,
					hasTenantPath: hasTenantPath,
				}
			}
		}
	}
	return m
}

// transformApiTs rewrites every API method to attach a `resources` property via
// Object.assign, enabling runtime resource-scope checks.
//
// Before:
//
//	methodName = (args) =>
//	  this.request({...});
//
// After:
//
//	methodName = Object.assign((args) =>
//	  this.request({...}), { resources: new Set<string>(["tenant"]) });
func transformApiTs(content string, metadataMap map[string]operationMetadata) (result string, total, withResources, withTenantPath int) {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Method-level JSDoc blocks start at exactly 2-space indent.
		// Query-property JSDoc blocks are at 6+ spaces — not matched here.
		if line != "  /**" {
			out = append(out, line)
			i++
			continue
		}

		jsdocLines, requestKey, advance := collectJSDoc(lines, i)
		i += advance

		methodLines, advance := collectMethodBody(lines, i)
		i += advance

		alreadyPatched := len(methodLines) > 0 && strings.Contains(methodLines[0], "Object.assign(")
		if requestKey != "" && !alreadyPatched {
			meta := metadataMap[requestKey]
			methodLines = transformMethod(methodLines, meta)
			total++
			if len(meta.resources) > 0 {
				withResources++
			}
			if meta.hasTenantPath {
				withTenantPath++
			}
		}

		out = append(out, jsdocLines...)
		out = append(out, methodLines...)
	}

	return strings.Join(out, "\n"), total, withResources, withTenantPath
}

// collectJSDoc returns the JSDoc lines, the @request key ("METHOD /path"), and
// the number of lines consumed (including the closing "*/").
func collectJSDoc(lines []string, start int) (jsdocLines []string, requestKey string, consumed int) {
	jsdocLines = []string{lines[start]}
	i := start + 1

	for i < len(lines) {
		l := lines[i]
		jsdocLines = append(jsdocLines, l)

		if strings.Contains(l, "@request ") {
			requestKey = parseRequestKey(l)
		}

		if strings.TrimSpace(l) == "*/" {
			i++
			break
		}
		i++
	}

	return jsdocLines, requestKey, i - start
}

// parseRequestKey extracts "METHOD /path" from a JSDoc @request line such as:
//
//   - @request GET:/api/v1/stable/tasks/{task}
func parseRequestKey(line string) string {
	fields := strings.Fields(line)
	for j, f := range fields {
		if f == "@request" && j+1 < len(fields) {
			req := fields[j+1]
			colonIdx := strings.Index(req, ":")
			if colonIdx > 0 {
				return req[:colonIdx] + " " + req[colonIdx+1:]
			}
		}
	}
	return ""
}

// collectMethodBody gathers lines from the current position until (and
// including) the method terminator "    });", returning the lines and the
// number of lines consumed.
func collectMethodBody(lines []string, start int) (methodLines []string, consumed int) {
	i := start
	for i < len(lines) {
		methodLines = append(methodLines, lines[i])
		if lines[i] == "    });" {
			i++
			break
		}
		i++
	}
	return methodLines, i - start
}

// transformMethod wraps the method assignment with Object.assign so that callers
// can check resource scope:
//
//	api.methodName.resources.has("tenant")
//
// It also injects xResources into the this.request({}) options so the value is
// available on the Axios config inside interceptors:
//
//	(config as any).xResources // ["tenant", "task"]
//
// When the OpenAPI path contains {tenant}, it injects xTenantId: tenant before
// ...params so explicit caller params.xTenantId can still override.
func transformMethod(lines []string, meta operationMetadata) []string {
	if len(lines) < 2 {
		return lines
	}

	prefix, afterEq, ok := strings.Cut(lines[0], " = ")
	if !ok {
		return lines
	}

	arrayLit := arrayLiteral(meta.resources)
	middle := lines[1 : len(lines)-1]
	if meta.hasTenantPath {
		middle = injectXTenantId(middle)
	}

	result := make([]string, 0, len(lines)+2)
	result = append(result, prefix+" = Object.assign("+afterEq)
	result = append(result, middle...)
	result = append(result, "      xResources: "+arrayLit+",")
	result = append(result, "    }), { resources: new Set<string>("+arrayLit+") });")

	return result
}

func injectXTenantId(lines []string) []string {
	for i, line := range lines {
		if strings.TrimSpace(line) == "...params," {
			out := make([]string, 0, len(lines)+1)
			out = append(out, lines[:i]...)
			out = append(out, "      xTenantId: tenant,")
			out = append(out, lines[i:]...)
			return out
		}
	}
	return lines
}

// arrayLiteral builds a TypeScript array literal from a slice of strings:
//
//	["tenant", "task"]
func arrayLiteral(resources []string) string {
	quoted := make([]string, len(resources))
	for i, r := range resources {
		quoted[i] = `"` + r + `"`
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

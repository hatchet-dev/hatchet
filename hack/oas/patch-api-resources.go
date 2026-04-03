// patch-api-resources patches the generated Api.ts file to attach x-resources
// metadata from the OpenAPI spec to each API method, enabling callers to check
// resource scope at runtime:
//
//	api.v1TaskGet.resources.has("tenant") // true
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

	// Build lookup: "METHOD /path" -> resources (may be nil/empty).
	resourcesMap := buildResourcesMap(spec)

	// Read, transform, write Api.ts.
	apiData, err := os.ReadFile(apiPath)
	if err != nil {
		log.Fatalf("reading %s: %v", apiPath, err)
	}

	result, total, withResources := transformApiTs(string(apiData), resourcesMap)

	if err := os.WriteFile(apiPath, []byte(result), 0644); err != nil { // nolint:gosec
		log.Fatalf("writing %s: %v", apiPath, err)
	}

	fmt.Printf("patched %s: %d methods transformed (%d with x-resources)\n", apiPath, total, withResources)
}

type resource struct {
	op     *Operation
	method string
}

// buildResourcesMap returns a map from "METHOD /path" to the x-resources list.
func buildResourcesMap(spec OpenAPISpec) map[string][]string {
	m := make(map[string][]string)
	for path, item := range spec.Paths {
		for _, entry := range []resource{
			{op: item.Get, method: "GET"},
			{op: item.Post, method: "POST"},
			{op: item.Put, method: "PUT"},
			{op: item.Patch, method: "PATCH"},
			{op: item.Delete, method: "DELETE"},
		} {
			if entry.op != nil {
				m[entry.method+" "+path] = entry.op.XResources
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
func transformApiTs(content string, resourcesMap map[string][]string) (result string, total, withResources int) {
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

		// Collect JSDoc lines and extract the @request annotation.
		jsdocLines, requestKey, advance := collectJSDoc(lines, i)
		i += advance

		// Collect the method body (everything up to and including "    });").
		methodLines, advance := collectMethodBody(lines, i)
		i += advance

		// Look up resources; transform regardless (empty Set for unscoped methods).
		// Skip if already patched (idempotent when run multiple times).
		alreadyPatched := len(methodLines) > 0 && strings.Contains(methodLines[0], "Object.assign(")
		if requestKey != "" && !alreadyPatched {
			res := resourcesMap[requestKey]
			methodLines = transformMethod(methodLines, res)
			total++
			if len(res) > 0 {
				withResources++
			}
		}

		out = append(out, jsdocLines...)
		out = append(out, methodLines...)
	}

	return strings.Join(out, "\n"), total, withResources
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
func transformMethod(lines []string, resources []string) []string {
	if len(lines) < 2 {
		return lines
	}

	// Split "  methodName = REST" into prefix and the part after " = ".
	prefix, afterEq, ok := strings.Cut(lines[0], " = ")
	if !ok {
		return lines // unexpected format; leave unchanged
	}

	arrayLit := arrayLiteral(resources)

	// Rebuild the lines:
	//   1. First line gets Object.assign( prepended.
	//   2. Middle lines are unchanged.
	//   3. A new xResources line is inserted before the closing    });
	//   4. The closing    }); becomes    }), { resources: new Set<string>([...]) });
	result := make([]string, 0, len(lines)+1)
	result = append(result, prefix+" = Object.assign("+afterEq)
	result = append(result, lines[1:len(lines)-1]...)
	result = append(result, "      xResources: "+arrayLit+",")
	result = append(result, "    }), { resources: new Set<string>("+arrayLit+") });")

	return result
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

package authz

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

var payloadFieldsNeverPopulated = map[string]string{
	"V1TaskEventList": "ToTaskRunEventMany never sets V1TaskEvent.output",
}

func TestEveryOperationDeclaresPayloadVisibility(t *testing.T) {
	spec, err := gen.GetSwagger()
	if err != nil {
		t.Fatal(err)
	}

	gated := handlersCallingCanViewPayloads(t)

	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			name := operationMethodName(op.OperationID)

			declared, ok := op.Extensions["x-payloads"].(bool)
			if !ok {
				t.Errorf("%s %s (%s): missing `x-payloads: true|false` declaration", method, path, op.OperationID)
				continue
			}

			reaches := reachesPayloadField(op)

			switch {
			case declared && !gated[name]:
				t.Errorf("%s declares x-payloads: true but its handler never calls authz.CanViewPayloads", name)
			case !declared && reaches != "" && payloadFieldsNeverPopulated[name] == "":
				t.Errorf("%s declares x-payloads: false but its response reaches %s; gate it with authz.CanViewPayloads and set x-payloads: true, or add it to payloadFieldsNeverPopulated", name, reaches)
			case !declared && reaches == "" && payloadFieldsNeverPopulated[name] != "":
				t.Errorf("%s is listed in payloadFieldsNeverPopulated but no x-payload property is reachable; remove the entry", name)
			}
		}
	}
}

func operationMethodName(operationId string) string {
	var b strings.Builder
	upper := true
	for _, r := range operationId {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			upper = true
			continue
		}
		if upper {
			r = unicode.ToUpper(r)
			upper = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

func handlersCallingCanViewPayloads(t *testing.T) map[string]bool {
	t.Helper()

	gated := map[string]bool{}
	fset := token.NewFileSet()

	err := filepath.WalkDir("../handlers", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "CanViewPayloads" {
					return true
				}
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "authz" {
					gated[fn.Name.Name] = true
				}
				return true
			})
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	return gated
}

func reachesPayloadField(op *openapi3.Operation) string {
	seen := map[*openapi3.Schema]bool{}

	for code, resp := range op.Responses.Map() {
		if !strings.HasPrefix(code, "2") || resp.Value == nil {
			continue
		}
		for _, mt := range resp.Value.Content {
			if hit := findPayloadField(mt.Schema, "", seen); hit != "" {
				return hit
			}
		}
	}

	return ""
}

func findPayloadField(ref *openapi3.SchemaRef, name string, seen map[*openapi3.Schema]bool) string {
	if ref == nil || ref.Value == nil || seen[ref.Value] {
		return ""
	}
	seen[ref.Value] = true

	if ref.Ref != "" {
		name = strings.TrimPrefix(ref.Ref, "#/components/schemas/")
	}

	s := ref.Value

	for prop, p := range s.Properties {
		if p.Value != nil {
			if marked, _ := p.Value.Extensions["x-payload"].(bool); marked {
				return name + "." + prop
			}
		}
		if hit := findPayloadField(p, name+"."+prop, seen); hit != "" {
			return hit
		}
	}

	children := []*openapi3.SchemaRef{s.Items}
	children = append(children, s.AllOf...)
	children = append(children, s.OneOf...)
	children = append(children, s.AnyOf...)
	if s.AdditionalProperties.Schema != nil {
		children = append(children, s.AdditionalProperties.Schema)
	}

	for _, c := range children {
		if hit := findPayloadField(c, name, seen); hit != "" {
			return hit
		}
	}

	return ""
}

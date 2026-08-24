// Command generator emits the Go SDK reference docs (fumadocs .mdx pages) from the
// SDK's exported API and doc comments.
//
// It is run from sdks/go via `go run ./docs/generator` (wired up as the
// `generate-sdk-docs-go` task) and writes into
// frontend/docs/content/docs/reference/go/.
//
// Ownership: the generator owns every .mdx file in that directory as well as
// feature-clients/meta.json. The top-level go/meta.json is merged rather than
// overwritten: existing entry order and separator strings are preserved, stale
// pages are dropped, and newly emitted pages are appended. The top-level
// reference/meta.json is never touched.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	sdkDir, err := os.Getwd()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(sdkDir, "hatchet.go")); err != nil {
		return fmt.Errorf("run from sdks/go (hatchet.go not found in %s)", sdkDir)
	}

	repoRoot, err := findRepoRoot(sdkDir)
	if err != nil {
		return err
	}
	outDir := filepath.Join(repoRoot, "frontend", "docs", "content", "docs", "reference", "go")

	root, err := loadPackage(sdkDir, "github.com/hatchet-dev/hatchet/sdks/go")
	if err != nil {
		return err
	}
	features, err := loadPackage(filepath.Join(sdkDir, "features"), "github.com/hatchet-dev/hatchet/sdks/go/features")
	if err != nil {
		return err
	}
	worker, err := loadPackage(filepath.Join(repoRoot, "pkg", "worker"), "github.com/hatchet-dev/hatchet/pkg/worker")
	if err != nil {
		return err
	}

	accessors := featureAccessors(root.pkg)

	if err := cleanOutDir(outDir); err != nil {
		return err
	}

	files := map[string]string{
		"client.mdx":    renderClientPage(root, accessors),
		"context.mdx":   renderContextPage(worker),
		"runnables.mdx": renderRunnablesPage(root),
	}
	for _, a := range accessors {
		files[filepath.Join("feature-clients", a.page+".mdx")] = renderFeaturePage(features, a)
	}
	files[filepath.Join("feature-clients", "meta.json")] = renderFeatureMeta(accessors)

	topPages := []string{"feature-clients"}
	for name := range files {
		if !strings.Contains(name, string(filepath.Separator)) && strings.HasSuffix(name, ".mdx") {
			topPages = append(topPages, strings.TrimSuffix(name, ".mdx"))
		}
	}
	topMeta, err := mergeTopMeta(filepath.Join(outDir, "meta.json"), topPages)
	if err != nil {
		return err
	}
	files["meta.json"] = topMeta

	if err := verifyMetaCoverage(files); err != nil {
		return err
	}

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(outDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		content := strings.TrimRight(files[name], "\n") + "\n"
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return err
		}
		fmt.Println("wrote", path)
	}

	return runPrettier(repoRoot, outDir)
}

// runPrettier formats the emitted files with the docs repo's own prettier so the
// generated output is byte-stable under the repo's formatting checks. Skipping it
// silently would emit unformatted docs that churn CI, so a missing prettier is fatal.
func runPrettier(repoRoot, outDir string) error {
	bin := filepath.Join(repoRoot, "frontend", "docs", "node_modules", ".bin", "prettier")
	if _, err := os.Stat(bin); err != nil {
		return fmt.Errorf("prettier not found at %s: run `pnpm install` in frontend/docs first", bin)
	}
	cmd := exec.Command(bin, "--write", "--log-level", "warn", outDir)
	cmd.Dir = filepath.Join(repoRoot, "frontend", "docs")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("prettier formatting pass failed: %w", err)
	}
	return nil
}

func findRepoRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "frontend", "docs")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate repo root (frontend/docs + go.mod) above %s", start)
		}
		dir = parent
	}
}

// cleanOutDir removes generator-owned files (all .mdx plus feature-clients/) so stale
// pages don't survive renames. The hand-maintained go/meta.json is preserved.
func cleanOutDir(outDir string) error {
	entries, err := os.ReadDir(outDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() == "feature-clients" {
			if err := os.RemoveAll(filepath.Join(outDir, e.Name())); err != nil {
				return err
			}
			continue
		}
		if strings.HasSuffix(e.Name(), ".mdx") {
			if err := os.Remove(filepath.Join(outDir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// mergeTopMeta merges the emitted top-level pages into the existing go/meta.json:
// existing entry order, separator strings ("---...---"), and non-string entries are
// preserved; string entries whose page was not emitted are dropped; emitted pages not
// already listed are appended in sorted order.
func mergeTopMeta(path string, emitted []string) (string, error) {
	meta := map[string]any{"title": "Go SDK"}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &meta); err != nil {
			return "", fmt.Errorf("parse %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	pages, _ := meta["pages"].([]any)

	exists := map[string]bool{}
	for _, p := range emitted {
		exists[p] = true
	}

	merged := []any{}
	present := map[string]bool{}
	for _, entry := range pages {
		s, isString := entry.(string)
		isSeparator := isString && strings.HasPrefix(s, "---") && strings.HasSuffix(s, "---")
		if isString && !isSeparator && !exists[s] {
			fmt.Println("meta.json: dropping stale page", s)
			continue
		}
		merged = append(merged, entry)
		if isString {
			present[s] = true
		}
	}

	var missing []string
	for _, p := range emitted {
		if !present[p] {
			missing = append(missing, p)
		}
	}
	sort.Strings(missing)
	for _, p := range missing {
		merged = append(merged, p)
	}

	meta["pages"] = merged
	out, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out) + "\n", nil
}

// verifyMetaCoverage asserts that every emitted .mdx page is reachable from a
// meta.json pages array, so a generator bug can't silently orphan a page.
func verifyMetaCoverage(files map[string]string) error {
	pagesOf := func(name string) (map[string]bool, error) {
		var m struct {
			Pages []any `json:"pages"`
		}
		if err := json.Unmarshal([]byte(files[name]), &m); err != nil {
			return nil, fmt.Errorf("parse emitted %s: %w", name, err)
		}
		set := map[string]bool{}
		for _, e := range m.Pages {
			if s, ok := e.(string); ok {
				set[s] = true
			}
		}
		return set, nil
	}
	top, err := pagesOf("meta.json")
	if err != nil {
		return err
	}
	feature, err := pagesOf(filepath.Join("feature-clients", "meta.json"))
	if err != nil {
		return err
	}

	var orphans []string
	for name := range files {
		if !strings.HasSuffix(name, ".mdx") {
			continue
		}
		dir, base := filepath.Split(name)
		page := strings.TrimSuffix(base, ".mdx")
		reachable := false
		switch dir {
		case "":
			reachable = top[page]
		case "feature-clients" + string(filepath.Separator):
			reachable = feature[page] && top["feature-clients"]
		}
		if !reachable {
			orphans = append(orphans, name)
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		return fmt.Errorf("emitted pages not reachable from any meta.json pages array: %s", strings.Join(orphans, ", "))
	}
	return nil
}

// pkgDocs bundles a parsed package with the per-file declaration index used to group
// feature-client pages by source file.
type pkgDocs struct {
	pkg       *doc.Package
	fset      *token.FileSet
	fileDecls map[string][]string // base file name -> top-level exported decl names, in declaration order
}

func loadPackage(dir, importPath string) (*pkgDocs, error) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var fileNames []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		fileNames = append(fileNames, e.Name())
	}
	sort.Strings(fileNames)

	var files []*ast.File
	fileDecls := map[string][]string{}
	for _, name := range fileNames {
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.IsExported() {
					fileDecls[name] = append(fileDecls[name], d.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.IsExported() {
						fileDecls[name] = append(fileDecls[name], ts.Name.Name)
					}
				}
			}
		}
	}

	p, err := doc.NewFromFiles(fset, files, importPath)
	if err != nil {
		return nil, err
	}
	return &pkgDocs{pkg: p, fset: fset, fileDecls: fileDecls}, nil
}

// ---------- doc comment helpers ----------

func skipDoc(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(text, "Deprecated:") || strings.Contains(lower, "internal use")
}

func mdDoc(p *pkgDocs, text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	printer := p.pkg.Printer()
	printer.HeadingLevel = 4
	md := string(printer.Markdown(p.pkg.Parser().Parse(text)))
	return strings.TrimRight(fenceIndentedCode(md), "\n") + "\n"
}

// fenceIndentedCode converts tab-indented code blocks (as emitted by
// go/doc/comment's Markdown printer) into fenced ```go blocks, since MDX does not
// support indented code blocks.
func fenceIndentedCode(md string) string {
	lines := strings.Split(md, "\n")
	var out []string
	i := 0
	for i < len(lines) {
		if !strings.HasPrefix(lines[i], "\t") {
			out = append(out, lines[i])
			i++
			continue
		}
		var block []string
		for i < len(lines) && (strings.HasPrefix(lines[i], "\t") || strings.TrimSpace(lines[i]) == "") {
			block = append(block, strings.TrimPrefix(lines[i], "\t"))
			i++
		}
		for len(block) > 0 && strings.TrimSpace(block[len(block)-1]) == "" {
			block = block[:len(block)-1]
		}
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, "```go")
		out = append(out, block...)
		out = append(out, "```", "")
	}
	return strings.Join(out, "\n")
}

func synopsis(p *pkgDocs, text string) string {
	s := p.pkg.Synopsis(text)
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
}

// ---------- signature helpers ----------

func exprString(e ast.Expr) string {
	// Normalize the SDK's internal import alias for the v0 client package.
	return strings.ReplaceAll(types.ExprString(e), "v0Client.", "client.")
}

type param struct {
	name string
	typ  string
}

func fieldListParams(fl *ast.FieldList) []param {
	if fl == nil {
		return nil
	}
	var out []param
	for _, f := range fl.List {
		typ := exprString(f.Type)
		if len(f.Names) == 0 {
			out = append(out, param{name: "", typ: typ})
			continue
		}
		for _, n := range f.Names {
			out = append(out, param{name: n.Name, typ: typ})
		}
	}
	return out
}

func signature(recv string, name string, ft *ast.FuncType) string {
	var b strings.Builder
	b.WriteString("func ")
	if recv != "" {
		b.WriteString("(" + recv + ") ")
	}
	b.WriteString(name)
	b.WriteString(renderFieldList(ft.Params, true))
	results := fieldListParams(ft.Results)
	switch len(results) {
	case 0:
	case 1:
		if results[0].name == "" {
			b.WriteString(" " + results[0].typ)
			break
		}
		fallthrough
	default:
		b.WriteString(" " + renderFieldList(ft.Results, false))
	}
	return b.String()
}

func renderFieldList(fl *ast.FieldList, forceParens bool) string {
	if fl == nil {
		if forceParens {
			return "()"
		}
		return ""
	}
	var parts []string
	for _, f := range fl.List {
		typ := exprString(f.Type)
		if len(f.Names) == 0 {
			parts = append(parts, typ)
			continue
		}
		names := make([]string, len(f.Names))
		for i, n := range f.Names {
			names[i] = n.Name
		}
		parts = append(parts, strings.Join(names, ", ")+" "+typ)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func recvString(fn *doc.Func) string {
	if fn.Decl.Recv == nil || len(fn.Decl.Recv.List) == 0 {
		return ""
	}
	f := fn.Decl.Recv.List[0]
	typ := exprString(f.Type)
	if len(f.Names) > 0 {
		return f.Names[0].Name + " " + typ
	}
	return typ
}

// ---------- markdown table helpers ----------

func escapeCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.ReplaceAll(s, "\n", " ")
}

func table(header []string, rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	widths := make([]int, len(header))
	for i, h := range header {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if w := utf8.RuneCountInString(cell); i < len(widths) && w > widths[i] {
				widths[i] = w
			}
		}
	}
	pad := func(s string, w int) string {
		return s + strings.Repeat(" ", w-utf8.RuneCountInString(s))
	}
	var b strings.Builder
	b.WriteString("|")
	for i, h := range header {
		b.WriteString(" " + pad(h, widths[i]) + " |")
	}
	b.WriteString("\n|")
	for i := range header {
		b.WriteString(" " + strings.Repeat("-", widths[i]) + " |")
	}
	b.WriteString("\n")
	for _, row := range rows {
		b.WriteString("|")
		for i := range header {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			b.WriteString(" " + pad(cell, widths[i]) + " |")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func paramsTable(ft *ast.FuncType) string {
	params := fieldListParams(ft.Params)
	if len(params) == 0 {
		return ""
	}
	rows := make([][]string, 0, len(params))
	for _, p := range params {
		name := p.name
		if name == "" {
			name = "_"
		}
		rows = append(rows, []string{"`" + name + "`", "`" + escapeCell(p.typ) + "`"})
	}
	return "Parameters:\n\n" + table([]string{"Name", "Type"}, rows)
}

func returnsTable(ft *ast.FuncType) string {
	results := fieldListParams(ft.Results)
	if len(results) == 0 {
		return ""
	}
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		rows = append(rows, []string{"`" + escapeCell(r.typ) + "`"})
	}
	return "Returns:\n\n" + table([]string{"Type"}, rows)
}

// ---------- section rendering ----------

func writeFuncSection(b *strings.Builder, p *pkgDocs, fn *doc.Func, level int) {
	heading := strings.Repeat("#", level)
	fmt.Fprintf(b, "%s `%s`\n\n", heading, fn.Name)
	if d := mdDoc(p, fn.Doc); d != "" {
		b.WriteString(d + "\n")
	}
	fmt.Fprintf(b, "```go\n%s\n```\n\n", signature(recvString(fn), fn.Name, fn.Decl.Type))
	if t := paramsTable(fn.Decl.Type); t != "" {
		b.WriteString(t + "\n")
	}
	if t := returnsTable(fn.Decl.Type); t != "" {
		b.WriteString(t + "\n")
	}
}

func methodRows(p *pkgDocs, fns []*doc.Func) [][]string {
	var rows [][]string
	for _, fn := range fns {
		if skipDoc(fn.Doc) {
			continue
		}
		rows = append(rows, []string{"`" + fn.Name + "`", escapeCell(synopsis(p, fn.Doc))})
	}
	return rows
}

func structFieldsTable(p *pkgDocs, st *ast.StructType) string {
	var rows [][]string
	for _, f := range st.Fields.List {
		docText := f.Doc.Text()
		if docText == "" && f.Comment != nil {
			docText = f.Comment.Text()
		}
		if skipDoc(docText) {
			continue
		}
		desc := escapeCell(strings.TrimSpace(strings.ReplaceAll(docText, "\n", " ")))
		typ := "`" + escapeCell(exprString(f.Type)) + "`"
		if len(f.Names) == 0 {
			rows = append(rows, []string{"`" + escapeCell(exprString(f.Type)) + "`", typ, desc})
			continue
		}
		for _, n := range f.Names {
			if !n.IsExported() {
				continue
			}
			rows = append(rows, []string{"`" + n.Name + "`", typ, desc})
		}
	}
	if len(rows) == 0 {
		return ""
	}
	return "Fields:\n\n" + table([]string{"Name", "Type", "Description"}, rows)
}

func typeSpec(t *doc.Type) *ast.TypeSpec {
	for _, spec := range t.Decl.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == t.Name {
			return ts
		}
	}
	return nil
}

// interfaceMethods returns the exported, non-embedded methods of an interface type.
func interfaceMethods(t *doc.Type) []*ast.Field {
	ts := typeSpec(t)
	if ts == nil {
		return nil
	}
	it, ok := ts.Type.(*ast.InterfaceType)
	if !ok {
		return nil
	}
	var out []*ast.Field
	for _, f := range it.Methods.List {
		if len(f.Names) == 0 || !f.Names[0].IsExported() {
			continue
		}
		if _, ok := f.Type.(*ast.FuncType); !ok {
			continue
		}
		out = append(out, f)
	}
	return out
}

func writeInterfaceMethodSection(b *strings.Builder, p *pkgDocs, f *ast.Field, level int) {
	name := f.Names[0].Name
	ft := f.Type.(*ast.FuncType)
	heading := strings.Repeat("#", level)
	fmt.Fprintf(b, "%s `%s`\n\n", heading, name)
	docText := f.Doc.Text()
	if docText == "" && f.Comment != nil {
		docText = f.Comment.Text()
	}
	if d := mdDoc(p, docText); d != "" {
		b.WriteString(d + "\n")
	}
	fmt.Fprintf(b, "```go\n%s\n```\n\n", signature("", name, ft))
	if t := paramsTable(ft); t != "" {
		b.WriteString(t + "\n")
	}
	if t := returnsTable(ft); t != "" {
		b.WriteString(t + "\n")
	}
}

// writeTypeSection renders a type: doc comment, associated consts/vars, struct fields,
// a methods summary table, and detailed sections for associated funcs and methods.
// skipCtors omits associated New* constructor functions (used on feature-client pages,
// where clients are obtained from the Client accessors instead).
func writeTypeSection(b *strings.Builder, p *pkgDocs, t *doc.Type, level int, title string, skipCtors bool) {
	if title == "" {
		title = t.Name
	}
	heading := strings.Repeat("#", level)
	fmt.Fprintf(b, "%s %s\n\n", heading, title)
	if d := mdDoc(p, t.Doc); d != "" {
		b.WriteString(d + "\n")
	}

	for _, c := range t.Consts {
		if skipDoc(c.Doc) {
			continue
		}
		fmt.Fprintf(b, "```go\n%s\n```\n\n", declString(p, c.Decl))
	}
	for _, v := range t.Vars {
		if skipDoc(v.Doc) {
			continue
		}
		fmt.Fprintf(b, "```go\n%s\n```\n\n", declString(p, v.Decl))
	}

	if ts := typeSpec(t); ts != nil {
		if st, ok := ts.Type.(*ast.StructType); ok {
			if ft := structFieldsTable(p, st); ft != "" {
				b.WriteString(ft + "\n")
			}
		}
	}

	var fns []*doc.Func
	for _, fn := range t.Funcs {
		if skipDoc(fn.Doc) || (skipCtors && strings.HasPrefix(fn.Name, "New")) {
			continue
		}
		fns = append(fns, fn)
	}
	var methods []*doc.Func
	for _, m := range t.Methods {
		if !skipDoc(m.Doc) {
			methods = append(methods, m)
		}
	}

	if len(methods) > 1 {
		b.WriteString("Methods:\n\n")
		b.WriteString(table([]string{"Name", "Description"}, methodRows(p, methods)) + "\n")
	}
	if len(fns)+len(methods) > 0 {
		fmt.Fprintf(b, "%s Functions\n\n", strings.Repeat("#", level+1))
	}
	for _, fn := range fns {
		writeFuncSection(b, p, fn, level+2)
	}
	for _, m := range methods {
		writeFuncSection(b, p, m, level+2)
	}
}

func declString(p *pkgDocs, decl *ast.GenDecl) string {
	// Render the declaration without doc comments.
	stripped := *decl
	stripped.Doc = nil
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, p.fset, &stripped); err != nil {
		return ""
	}
	return strings.TrimRight(buf.String(), "\n")
}

// ---------- feature client discovery ----------

type accessor struct {
	method   string // method name on Client, e.g. "RateLimits"
	typeName string // type name in the features package, e.g. "RateLimitsClient"
	page     string // page slug, e.g. "ratelimits"
	title    string // page title, e.g. "Rate Limits"
	doc      string // accessor doc comment from client.go
}

// featureAccessors finds Client methods returning *features.<T> and maps each to a
// feature-clients page.
func featureAccessors(p *doc.Package) []accessor {
	var out []accessor
	for _, t := range p.Types {
		if t.Name != "Client" {
			continue
		}
		for _, m := range t.Methods {
			if m.Decl.Type.Results == nil || len(m.Decl.Type.Results.List) != 1 {
				continue
			}
			star, ok := m.Decl.Type.Results.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			sel, ok := star.X.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			if ident, ok := sel.X.(*ast.Ident); !ok || ident.Name != "features" {
				continue
			}
			out = append(out, accessor{
				method:   m.Name,
				typeName: sel.Sel.Name,
				page:     strings.ToLower(m.Name),
				title:    splitCamel(m.Name),
				doc:      m.Doc,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].page < out[j].page })
	return out
}

// splitCamel converts "RateLimits" to "Rate Limits", preserving acronyms ("CEL").
func splitCamel(s string) string {
	var words []string
	var cur []rune
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prevLower := unicode.IsLower(runes[i-1])
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if prevLower || (unicode.IsUpper(runes[i-1]) && nextLower) {
				words = append(words, string(cur))
				cur = nil
			}
		}
		cur = append(cur, r)
	}
	if len(cur) > 0 {
		words = append(words, string(cur))
	}
	return strings.Join(words, " ")
}

// ---------- pages ----------

func frontmatter(title string) string {
	return fmt.Sprintf("---\ntitle: \"%s\"\n---\n\n", title)
}

// No generated-file banner is emitted: JSX comments are banned in this repo's
// MDX (prettier corrupts them, see .claude/skills/sdk-reference-docs/SKILL.md)
// and MDX has no other comment syntax. Ownership is documented in the skill and
// enforced by regeneration.
func generatedNotice() string {
	return ""
}

func findType(p *pkgDocs, name string) *doc.Type {
	for _, t := range p.pkg.Types {
		if t.Name == name {
			return t
		}
	}
	return nil
}

// funcsReturning returns package-level functions whose only result type is named typeName.
func funcsReturning(p *pkgDocs, typeName string) []*doc.Func {
	t := findType(p, typeName)
	if t == nil {
		return nil
	}
	var out []*doc.Func
	for _, fn := range t.Funcs {
		if skipDoc(fn.Doc) {
			continue
		}
		out = append(out, fn)
	}
	return out
}

func optionTable(p *pkgDocs, fns []*doc.Func) string {
	var rows [][]string
	for _, fn := range fns {
		sig := fn.Name + renderFieldList(fn.Decl.Type.Params, true)
		rows = append(rows, []string{"`" + fn.Name + "`", "`" + escapeCell(sig) + "`", escapeCell(synopsis(p, fn.Doc))})
	}
	return table([]string{"Name", "Signature", "Description"}, rows)
}

const clientIntro = `This is the Go SDK reference, documenting methods available for interacting with Hatchet resources. Check out the [user guide](/v1) for an introduction for getting your first tasks running. For complete, generated API documentation, see the [Go package docs on pkg.go.dev](https://pkg.go.dev/github.com/hatchet-dev/hatchet/sdks/go).

By default, the client reads its configuration (token, host, TLS settings, and so on) from the ` + "`HATCHET_CLIENT_*`" + ` environment variables. Configuration can be overridden with client options from the ` + "`github.com/hatchet-dev/hatchet/pkg/client`" + ` package, such as ` + "`WithToken`" + `, ` + "`WithHostPort`" + `, and ` + "`WithNamespace`" + `.
`

func renderClientPage(p *pkgDocs, accessors []accessor) string {
	var b strings.Builder
	b.WriteString(frontmatter("Client"))
	b.WriteString(generatedNotice())
	b.WriteString("# Hatchet Go SDK Reference\n\n")
	b.WriteString(clientIntro + "\n")

	accessorNames := map[string]accessor{}
	for _, a := range accessors {
		accessorNames[a.method] = a
	}

	if t := findType(p, "Client"); t != nil {
		fmt.Fprintf(&b, "## Client\n\n")
		if d := mdDoc(p, t.Doc); d != "" {
			b.WriteString(d + "\n")
		}

		var ctor, methods []*doc.Func
		for _, fn := range t.Funcs {
			if !skipDoc(fn.Doc) {
				ctor = append(ctor, fn)
			}
		}
		for _, m := range t.Methods {
			if skipDoc(m.Doc) {
				continue
			}
			if _, isAccessor := accessorNames[m.Name]; isAccessor {
				continue
			}
			methods = append(methods, m)
		}

		b.WriteString("Methods:\n\n")
		b.WriteString(table([]string{"Name", "Description"}, methodRows(p, methods)) + "\n")

		b.WriteString("### Feature clients\n\n")
		b.WriteString("The client exposes lazily-initialized [feature clients](./feature-clients/" + accessors[0].page + ") as methods:\n\n")
		for _, a := range accessors {
			fmt.Fprintf(&b, "#### `%s()`\n\n", a.method)
			desc := synopsis(p, a.doc)
			fmt.Fprintf(&b, "%s See the [%s client](./feature-clients/%s).\n\n", desc, a.title, a.page)
		}

		b.WriteString("### Functions\n\n")
		for _, fn := range ctor {
			writeFuncSection(&b, p, fn, 4)
		}
		for _, m := range methods {
			writeFuncSection(&b, p, m, 4)
		}
	}

	if t := findType(p, "Worker"); t != nil {
		writeTypeSection(&b, p, t, 2, "Worker", false)
	}
	if opts := funcsReturning(p, "WorkerOption"); len(opts) > 0 {
		b.WriteString("## Worker options\n\n")
		b.WriteString("Options for `Client.NewWorker`:\n\n")
		b.WriteString(optionTable(p, opts) + "\n")
	}

	return b.String()
}

const contextIntro = `The Hatchet ` + "`Context`" + ` provides helper methods and useful data to tasks at runtime. It is passed as the first argument to all task functions.

There are two context types you'll encounter:

- ` + "`hatchet.Context`" + ` - The standard context for regular tasks, with methods for logging, parent output retrieval, streaming, and more. It embeds Go's ` + "`context.Context`" + `, so it can be passed anywhere a standard context is expected (including as the ` + "`ctx`" + ` argument to the run methods when spawning child workflows from inside a task).
- ` + "`hatchet.DurableContext`" + ` - An extended context for durable tasks that adds methods for durable execution like ` + "`SleepFor`" + `, ` + "`WaitForEvent`" + `, and ` + "`Memo`" + `. Durable waits are "global": they wait in real time and survive transient failures like worker restarts.

Results of durable waits (` + "`*SingleWaitResult`" + `, ` + "`*WaitResult`" + `) expose ` + "`Unmarshal`" + ` methods to decode the matched payload; ` + "`hatchet.EventInto(event, &dest)`" + ` is a convenience wrapper for the single-event case.
`

func renderContextPage(worker *pkgDocs) string {
	var b strings.Builder
	b.WriteString(frontmatter("Context"))
	b.WriteString(generatedNotice())
	b.WriteString("# Context\n\n")
	b.WriteString(contextIntro + "\n")

	sections := []struct {
		typeName string
		title    string
	}{
		{"HatchetContext", "Context"},
		{"DurableHatchetContext", "DurableContext"},
	}
	for _, s := range sections {
		t := findType(worker, s.typeName)
		if t == nil {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n", s.title)
		methods := interfaceMethods(t)
		var rows [][]string
		for _, f := range methods {
			docText := f.Doc.Text()
			if docText == "" && f.Comment != nil {
				docText = f.Comment.Text()
			}
			if skipDoc(docText) {
				continue
			}
			rows = append(rows, []string{"`" + f.Names[0].Name + "`", escapeCell(synopsis(worker, docText))})
		}
		b.WriteString("Methods:\n\n")
		b.WriteString(table([]string{"Name", "Description"}, rows) + "\n")
		b.WriteString("### Functions\n\n")
		for _, f := range methods {
			docText := f.Doc.Text()
			if docText == "" && f.Comment != nil {
				docText = f.Comment.Text()
			}
			if skipDoc(docText) {
				continue
			}
			writeInterfaceMethodSection(&b, worker, f, 4)
		}
	}
	return b.String()
}

const runnablesIntro = `Runnables in the Hatchet Go SDK are things that can be run, namely tasks and workflows. The two main types you'll encounter are:

- ` + "`Workflow`" + `, which lets you declare tasks with ` + "`NewTask`" + ` and call the run methods
- ` + "`StandaloneTask`" + `, which is a single task returned by ` + "`client.NewStandaloneTask`" + ` (or its durable/batch variants) and supports the same run methods

Both implement the ` + "`WorkflowBase`" + ` interface and can be registered on a worker with ` + "`hatchet.WithWorkflows`" + `. See the [Client page](./client) for the constructors.
`

func renderRunnablesPage(p *pkgDocs) string {
	var b strings.Builder
	b.WriteString(frontmatter("Runnables"))
	b.WriteString(generatedNotice())
	b.WriteString("# Runnables\n\n")
	b.WriteString(runnablesIntro + "\n")

	pinned := []string{
		"Workflow",
		"StandaloneTask",
		"Task",
		"WorkflowRunRef",
		"WorkflowResult",
		"TaskResult",
		"RunManyOpt",
	}
	optionGroups := []struct {
		typeName string
		title    string
		intro    string
	}{
		{"WorkflowOption", "Workflow options", "Options for `Client.NewWorkflow` (and standalone task constructors):"},
		{"TaskOption", "Task options", "Options for `Workflow.NewTask` and the other task constructors:"},
		{"RunOptFunc", "Run options", "Options for the `Run`, `RunNoWait`, and `RunMany` methods:"},
	}

	rendered := map[string]bool{}
	skipTypes := map[string]bool{
		// Aliases and interfaces that are pure plumbing for the types above.
		"WorkflowBase":         true,
		"StandaloneTaskOption": true,
		"EmbeddedBackend":      true,
		// Documented on the Context page.
		"Context":        true,
		"DurableContext": true,
	}
	for _, g := range optionGroups {
		rendered[g.typeName] = true
	}

	for _, name := range pinned {
		if t := findType(p, name); t != nil && !skipDoc(t.Doc) {
			writeTypeSection(&b, p, t, 2, "", false)
			rendered[name] = true
		}
	}

	for _, g := range optionGroups {
		opts := funcsReturning(p, g.typeName)
		if len(opts) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", g.title, g.intro)
		b.WriteString(optionTable(p, opts) + "\n")
	}

	// Condition helpers: package-level functions returning condition.Condition.
	var condFns []*doc.Func
	for _, fn := range p.pkg.Funcs {
		if skipDoc(fn.Doc) {
			continue
		}
		results := fieldListParams(fn.Decl.Type.Results)
		if len(results) == 1 && results[0].typ == "condition.Condition" {
			condFns = append(condFns, fn)
		}
	}
	if len(condFns) > 0 {
		b.WriteString("## Conditions\n\n")
		b.WriteString("Helpers for building the conditions used with `WithWaitFor`, `WithSkipIf`, and `DurableContext.WaitFor` (see [Context](./context)):\n\n")
		b.WriteString(optionTable(p, condFns) + "\n")
	}

	// Remaining exported types (errors, configs, and so on), in sorted order.
	var rest []*doc.Type
	for _, t := range p.pkg.Types {
		if rendered[t.Name] || skipTypes[t.Name] || t.Name == "Client" || t.Name == "Worker" || t.Name == "WorkerOption" {
			continue
		}
		if skipDoc(t.Doc) {
			continue
		}
		rest = append(rest, t)
	}
	sort.Slice(rest, func(i, j int) bool { return rest[i].Name < rest[j].Name })
	if len(rest) > 0 {
		b.WriteString("## Other types\n\n")
		for _, t := range rest {
			writeTypeSection(&b, p, t, 3, "", false)
		}
	}

	// Remaining package-level functions (go/doc already associates constructors and
	// option builders with their types, so only true leftovers appear here).
	condNames := map[string]bool{}
	for _, fn := range condFns {
		condNames[fn.Name] = true
	}
	var restFns []*doc.Func
	for _, fn := range p.pkg.Funcs {
		if skipDoc(fn.Doc) || condNames[fn.Name] {
			continue
		}
		restFns = append(restFns, fn)
	}
	if len(restFns) > 0 {
		b.WriteString("## Other functions\n\n")
		for _, fn := range restFns {
			writeFuncSection(&b, p, fn, 3)
		}
	}

	return b.String()
}

func renderFeaturePage(features *pkgDocs, a accessor) string {
	var b strings.Builder
	b.WriteString(frontmatter(a.title))
	b.WriteString(generatedNotice())
	fmt.Fprintf(&b, "# %s Client\n\n", a.title)
	fmt.Fprintf(&b, "Accessed via `client.%s()` on the [Hatchet client](../client).\n\n", a.method)

	// The features package keeps one file per feature client; render the client type
	// first, then every other exported declaration from the same file.
	fileName := ""
	for name, decls := range features.fileDecls {
		for _, d := range decls {
			if d == a.typeName {
				fileName = name
			}
		}
	}

	if t := findType(features, a.typeName); t != nil {
		writeTypeSection(&b, features, t, 2, "", true)
	}

	if fileName != "" {
		var others []string
		for _, name := range features.fileDecls[fileName] {
			if name == a.typeName || strings.HasPrefix(name, "New") {
				continue
			}
			others = append(others, name)
		}
		var sections strings.Builder
		for _, name := range others {
			if t := findType(features, name); t != nil {
				if skipDoc(t.Doc) {
					continue
				}
				writeTypeSection(&sections, features, t, 3, "", true)
				continue
			}
			for _, fn := range features.pkg.Funcs {
				if fn.Name == name && !skipDoc(fn.Doc) {
					writeFuncSection(&sections, features, fn, 3)
				}
			}
		}
		if sections.Len() > 0 {
			b.WriteString("## Related types\n\n")
			b.WriteString(sections.String())
		}
	}

	return b.String()
}

func renderFeatureMeta(accessors []accessor) string {
	var b strings.Builder
	b.WriteString("{\n  \"pages\": [\n")
	for i, a := range accessors {
		comma := ","
		if i == len(accessors)-1 {
			comma = ""
		}
		fmt.Fprintf(&b, "    %q%s\n", a.page, comma)
	}
	b.WriteString("  ],\n  \"title\": \"Feature Clients\"\n}\n")
	return b.String()
}

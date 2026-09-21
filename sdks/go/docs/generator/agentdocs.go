package main

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The "Documentation for agents" block injected into the package doc comment in
// sdks/go/hatchet.go, so that pkg.go.dev (where coding agents land after `go get`)
// carries a compact index of the docs the agent needs. The docs site serves plain
// markdown for every page at <baseUrl><path>.md; the block lists those URLs.
//
// The generated region inside the doc comment starts at the agentDocsHeading line
// and runs to the end of the comment; it is terminated by the agentDocsDirective
// line, a Go comment directive that gofmt pins to the end of the doc comment and
// go/doc excludes from the rendered docs. The block is replaced in full on every
// run and created when absent. Content comes from the shared hand-maintained
// mapping frontend/docs/reference-map.json (agentDocs + featureClients).

const (
	agentDocsHeading   = "// # Documentation for agents"
	agentDocsDirective = "//hatchet:agent-docs-generated (do not edit; regenerate with `go run ./docs/generator`)"
	packageClauseLine  = "package hatchet"
)

// renderAgentDocsComment renders the doc-comment lines of the generated region
// (heading through directive), each already prefixed as a Go comment line. It
// hard-fails when a section link path does not exist as a docs content file.
func renderAgentDocsComment(m *refMap, repoRoot string) ([]string, error) {
	const lang = "go"
	d := m.AgentDocs
	if d.BaseURL == "" || d.Lead == "" || len(d.Sections) == 0 {
		return nil, fmt.Errorf("reference-map.json agentDocs is missing baseUrl, lead, or sections")
	}

	comment := func(text string) string {
		if text == "" {
			return "//"
		}
		return "// " + text
	}

	lines := []string{
		agentDocsHeading,
		comment(""),
		comment(d.Lead),
	}

	for _, section := range d.Sections {
		var items []string
		for _, link := range section.Links {
			if !linkAppliesTo(link, lang) {
				continue
			}
			if !contentPageExists(repoRoot, link.Path) {
				return nil, fmt.Errorf("reference-map.json agentDocs link %q (%s) does not exist under frontend/docs/content/docs", link.Title, link.Path)
			}
			items = append(items, fmt.Sprintf("  - %s: %s%s.md", link.Title, d.BaseURL, link.Path))
		}
		if len(items) == 0 {
			continue
		}
		lines = append(lines, comment(""), comment(section.Title+":"), comment(""))
		for _, item := range items {
			lines = append(lines, comment(item))
		}
	}

	lines = append(lines,
		comment(""),
		comment(fmt.Sprintf("Go SDK reference (overview: %s/reference/%s.md):", d.BaseURL, lang)),
		comment(""),
	)
	var concepts []string
	for concept := range m.FeatureClients {
		concepts = append(concepts, concept)
	}
	sort.Strings(concepts)
	for _, concept := range concepts {
		f := m.FeatureClients[concept]
		slug, ok := f.Slugs[lang]
		if !ok {
			continue
		}
		item := fmt.Sprintf("  - %s: %s/reference/%s/feature-clients/%s.md", f.Title, d.BaseURL, lang, slug)
		if f.Guide != nil {
			item += fmt.Sprintf(" (guide: %s%s.md)", d.BaseURL, *f.Guide)
		}
		lines = append(lines, comment(item))
	}

	lines = append(lines, comment(""), agentDocsDirective)
	return lines, nil
}

func linkAppliesTo(link agentDocsLink, lang string) bool {
	if len(link.Langs) == 0 {
		return true
	}
	for _, l := range link.Langs {
		if l == lang {
			return true
		}
	}
	return false
}

func contentPageExists(repoRoot, page string) bool {
	rel := strings.TrimPrefix(page, "/")
	contentDir := filepath.Join(repoRoot, "frontend", "docs", "content", "docs")
	if _, err := os.Stat(filepath.Join(contentDir, rel+".mdx")); err == nil {
		return true
	}
	_, err := os.Stat(filepath.Join(contentDir, rel, "index.mdx"))
	return err == nil
}

// updateAgentDocs splices the generated block into the package doc comment in
// sdks/go/hatchet.go: an existing region (agentDocsHeading through the line before
// the package clause) is replaced in full; otherwise the block is appended to the
// end of the doc comment. The result is passed through go/format so the emitted
// file is gofmt-canonical and byte-stable across runs.
func updateAgentDocs(sdkDir, repoRoot string, m *refMap) error {
	path := filepath.Join(sdkDir, "hatchet.go")
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(src), "\n")
	pkgIdx := -1
	headingIdx := -1
	for i, line := range lines {
		if line == packageClauseLine {
			pkgIdx = i
			break
		}
		if line == agentDocsHeading {
			headingIdx = i
		}
	}
	if pkgIdx == -1 {
		return fmt.Errorf("%s: package clause %q not found", path, packageClauseLine)
	}
	if pkgIdx == 0 || !strings.HasPrefix(lines[pkgIdx-1], "//") {
		return fmt.Errorf("%s: expected a doc comment immediately above the package clause", path)
	}

	block, err := renderAgentDocsComment(m, repoRoot)
	if err != nil {
		return err
	}

	var out []string
	if headingIdx != -1 {
		// Replace the existing region: heading line through the end of the comment.
		out = append(out, lines[:headingIdx]...)
	} else {
		// Append to the doc comment, separated from the existing text.
		out = append(out, lines[:pkgIdx]...)
		out = append(out, "//")
	}
	out = append(out, block...)
	out = append(out, lines[pkgIdx:]...)

	formatted, err := format.Source([]byte(strings.Join(out, "\n")))
	if err != nil {
		return fmt.Errorf("format %s after splicing agent docs: %w", path, err)
	}
	if string(formatted) == string(src) {
		return nil
	}
	if err := os.WriteFile(path, formatted, 0o600); err != nil { // #nosec G703 -- path is the generator's own working directory plus a fixed file name, not attacker-controlled
		return err
	}
	fmt.Println("wrote", path)
	return nil
}

package main

// Sanitization policy.
//
// Every input to this program is attacker controlled: process names come from
// /proc, SQL text from pg_stat_statements, symbol names from pprof, and the
// test binary or any of its dependencies can print whatever it likes into the
// summaries. The comment is made safe by construction, not by spotting bad
// input:
//
//  1. Outside fenced code blocks the comment holds only constant text plus,
//     in exactly these places, these values:
//     - numbers the program formatted itself from parsed floats and ints (num);
//     - sample times that matched timeRE;
//     - process names in Mermaid series labels and in the process list, after
//       safeName replaced any URL in them, reduced them to [A-Za-z0-9._:/+-],
//       cut them to nameMax characters and made sure they are not empty. The
//       process list also wraps each name in an inline code span;
//     - the matrix label, which must match labelRE or the program exits 1;
//     - the run URL, which must match runURLRE or is omitted, never rewritten.
//  2. Inside fenced blocks sanitizeBlock strips ANSI escapes, control
//     characters other than newline and tab, and bidi or zero-width
//     characters; turns every backtick into a single quote so no line can
//     close the fence; neutralizes tilde fences; redacts secret-shaped tokens
//     (secretRules, key names kept); replaces URLs with "[url removed]"; turns
//     @mentions into "(at)name"; and caps lines at lineMax characters and
//     blocks at blockMaxLines lines, noting truncation.
//  3. In Mermaid, labels and series names are quoted strings that after rule 1
//     cannot contain a quote, a newline or "]" (mermaidString guards this
//     again); every number goes through num, which maps NaN and Inf to 0.
//     Nothing else untrusted is ever interpolated into a chart.
//  4. The only HTML is the fixed <details><summary> wrapper with constant text.
//  5. If the comment exceeds -max-bytes, sections are dropped from the end
//     (pg_stat, pprof, the process chart) and then the remaining fenced blocks
//     are truncated; a fenced block is never cut without being closed.
//  6. Parsing is total: a malformed sample line is skipped and counted, never
//     trusted.

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	nameMax       = 40
	lineMax       = 400
	blockMaxLines = 200
	truncatedNote = "... (truncated)"
)

var (
	timeRE   = regexp.MustCompile(`^\d\d:\d\d:\d\d$`)
	labelRE  = regexp.MustCompile(`^[A-Za-z0-9 ,()._-]{1,120}$`)
	runURLRE = regexp.MustCompile(`^https://github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+/actions/runs/[0-9]+$`)

	// CSI sequences, OSC sequences up to BEL or ST, nF escapes such as ESC ( B,
	// and two-byte escapes.
	ansiRE       = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\)?|[ -/]+[0-~]|[@-Z\\-_])`)
	urlRE        = regexp.MustCompile(`(?i)(?:https?|ftp|file)://\S+|mailto:\S+|\bwww\.\S+`)
	mentionRE    = regexp.MustCompile(`@([A-Za-z0-9_])`)
	tildeFenceRE = regexp.MustCompile(`^\s*~~~`)
)

type secretRule struct {
	re   *regexp.Regexp
	repl string
}

// Order matters: the bearer rule runs before the generic key=value rule so
// "Authorization: Bearer x" loses x rather than the word Bearer.
var secretRules = []secretRule{
	{regexp.MustCompile(`(?:gh[posur]|github_pat)_[A-Za-z0-9_]{8,}`), "[redacted]"},
	{regexp.MustCompile(`AKIA[0-9A-Z]{16}`), "[redacted]"},
	{regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`), "[redacted]"},
	{regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]+`), "[redacted]"},
	{regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`), "[redacted]"},
	{regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`), "[redacted]"},
	{regexp.MustCompile(`(?i)(bearer\s+)\S+`), "${1}[redacted]"},
	{regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|authorization|bearer)(\s*[=:]\s*)\S+`), "${1}${2}[redacted]"},
}

// sanitizeBlock prepares untrusted text for a fenced code block and returns
// its lines, capped at blockMaxLines plus a truncation note.
func sanitizeBlock(text string) []string {
	text = ansiRE.ReplaceAllString(text, "")
	text = stripControls(text)
	text = strings.ReplaceAll(text, "`", "'")
	raw := strings.Split(strings.TrimRight(text, "\n"), "\n")
	out := make([]string, 0, len(raw))
	for i, line := range raw {
		if i == blockMaxLines {
			out = append(out, truncatedNote)
			break
		}
		out = append(out, sanitizeLine(line))
	}
	return out
}

func sanitizeLine(line string) string {
	if tildeFenceRE.MatchString(line) {
		line = strings.Replace(line, "~~~", "---", 1)
	}
	for _, rule := range secretRules {
		line = rule.re.ReplaceAllString(line, rule.repl)
	}
	line = urlRE.ReplaceAllString(line, "[url removed]")
	line = mentionRE.ReplaceAllString(line, "(at)${1}")
	if utf8.RuneCountInString(line) > lineMax {
		line = string([]rune(line)[:lineMax]) + " ..."
	}
	return line
}

// stripControls drops everything a terminal or a renderer could act on:
// C0 and C1 controls, and the Unicode bidi and zero-width characters that
// can visually reorder or hide text even inside a code block.
func stripControls(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t':
			return r
		case r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f):
			return -1
		case r == utf8.RuneError:
			return '?'
		case (r >= 0x200b && r <= 0x200f) || r == 0x2028 || r == 0x2029 ||
			(r >= 0x202a && r <= 0x202e) || (r >= 0x2060 && r <= 0x2064) ||
			(r >= 0x2066 && r <= 0x2069) || r == 0xfeff:
			return -1
		}
		return r
	}, s)
}

// safeName reduces a process name to the allowlist used outside code fences.
// Distinct raw names can collapse onto one safe name; their shares merge.
func safeName(raw string) string {
	raw = urlRE.ReplaceAllString(raw, "[url removed]")
	var b strings.Builder
	n := 0
	for _, r := range raw {
		if n == nameMax {
			break
		}
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			strings.ContainsRune("._:/+-", r)
		if !ok {
			r = '_'
		}
		b.WriteRune(r)
		n++
	}
	if b.Len() == 0 {
		return "_"
	}
	return b.String()
}

// mermaidString is the last guard before a quoted Mermaid string; callers
// already pass allowlisted names or matched times.
func mermaidString(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '"' || r == ']' || r == '\n' || r == '\r' || r == '\\' {
			return '_'
		}
		return r
	}, s)
}

// num formats every number that reaches the markdown or a chart.
func num(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		f = 0
	}
	return strconv.FormatFloat(f, 'f', 1, 64)
}

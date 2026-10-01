package main

import (
	"math"
	"strings"
	"testing"
)

func TestSanitizeBlockRules(t *testing.T) {
	cases := []struct {
		name, in string
		want     []string
	}{
		{"fence breakout and html stay inert", "SELECT 1; ```\n<img onerror=alert(1) src=x>\n~~~\n  ~~~~ x\n",
			[]string{"SELECT 1; '''", "<img onerror=alert(1) src=x>", "---", "  ---~ x"}},
		{"github tokens", "ghp_abcdefghijklmnop123 gho_abcdefghijklmnop ghs_abcdefghijklmnop github_pat_abcdefghijklmnop",
			[]string{"[redacted] [redacted] [redacted] [redacted]"}},
		{"aws openai slack jwt", "AKIAABCDEFGHIJKLMNOP sk-abcdefghijklmnopqrstuvwxyz xoxb-123-456 eyJhbGciOi.eyJzdWIi.SflKxw",
			[]string{"[redacted] [redacted] [redacted] [redacted]"}},
		{"private key header", "-----BEGIN RSA PRIVATE KEY-----", []string{"[redacted]"}},
		{"key value pairs keep the key", "password=hunter2 api_key: abc token = x access_token=y Secret:z",
			[]string{"password=[redacted] api_key: [redacted] token = [redacted] access_token=[redacted] Secret:[redacted]"}},
		{"bearer loses the token not the word", "Authorization: Bearer abc123",
			[]string{"Authorization: [redacted] [redacted]"}},
		{"ansi colour and osc", "\x1b[31mred\x1b[0m \x1b]0;title\x07x \x1b(Bq",
			[]string{"red x q"}},
		{"control chars except tab", "\x00a\x01b\tc\x7fd\r", []string{"ab\tcd"}},
		{"bidi and zero width", "a\u202eb\u200bc\ufeffd", []string{"abcd"}},
		{"invalid utf8", "a\xffb", []string{"a?b"}},
		{"urls", "see http://a.b/c https://x.y ftp://f/g file:///etc/passwd mailto:a@b.c www.z.w end",
			[]string{"see [url removed] [url removed] [url removed] [url removed] [url removed] [url removed] end"}},
		{"mentions", "@octocat hi a@b @> @_x", []string{"(at)octocat hi a(at)b @> (at)_x"}},
		{"backticks become quotes", "a `b` ``` c", []string{"a 'b' ''' c"}},
		{"empty", "", []string{""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitizeBlock(c.in)
			if strings.Join(got, "\n") != strings.Join(c.want, "\n") {
				t.Errorf("sanitizeBlock(%q)\n got %q\nwant %q", c.in, got, c.want)
			}
		})
	}
}

func TestSanitizeBlockCaps(t *testing.T) {
	long := strings.Repeat("x", 500)
	got := sanitizeBlock(long)
	if len(got) != 1 || got[0] != strings.Repeat("x", lineMax)+" ..." {
		t.Errorf("long line not capped: %d chars", len(got[0]))
	}

	lines := make([]string, 250)
	for i := range lines {
		lines[i] = "line"
	}
	got = sanitizeBlock(strings.Join(lines, "\n"))
	if len(got) != blockMaxLines+1 || got[blockMaxLines] != truncatedNote {
		t.Errorf("block not capped: %d lines, last %q", len(got), got[len(got)-1])
	}
	for _, l := range got {
		if strings.Contains(l, "`") {
			t.Errorf("backtick survived: %q", l)
		}
	}
}

func TestSafeName(t *testing.T) {
	cases := map[string]string{
		"`](https://evil.example)@octocat": "____url_removed_",
		"www.evil.example":                 "_url_removed_",
		"loadtest":                         "loadtest",
		"kworker/u8:1-events":              "kworker/u8:1-events",
		"a b\"c]d\ne":                      "a_b_c_d_e",
		"":                                 "_",
		"héllo\xff":                        "h_llo_",
		strings.Repeat("y", 50):            strings.Repeat("y", nameMax),
	}
	for in, want := range cases {
		if got := safeName(in); got != want {
			t.Errorf("safeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNum(t *testing.T) {
	cases := map[float64]string{math.NaN(): "0.0", math.Inf(1): "0.0", math.Inf(-1): "0.0", 12.345: "12.3", 100: "100.0"}
	for in, want := range cases {
		if got := num(in); got != want {
			t.Errorf("num(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestMermaidString(t *testing.T) {
	if got := mermaidString("a\"b]c\nd\\e"); got != "a_b_c_d_e" {
		t.Errorf("mermaidString = %q", got)
	}
}

func TestLabelAndRunURLPatterns(t *testing.T) {
	labels := map[string]bool{
		"load (optimistic true, race false)": true,
		"[link](x)":                          false,
		"":                                   false,
		"a`b":                                false,
		strings.Repeat("a", 121):             false,
	}
	for in, want := range labels {
		if got := labelRE.MatchString(in); got != want {
			t.Errorf("labelRE(%q) = %v, want %v", in, got, want)
		}
	}
	urls := map[string]bool{
		"https://github.com/hatchet-dev/hatchet/actions/runs/1":     true,
		"https://evil.example/hatchet-dev/hatchet/actions/runs/1":   false,
		"https://github.com@evil.example/hatchet/actions/runs/1":    false,
		"https://github.com/hatchet-dev/hatchet/actions/runs/1@x":   false,
		"https://github.com/hatchet-dev/hatchet/actions/runs/1?x=y": false,
		"http://github.com/hatchet-dev/hatchet/actions/runs/1":      false,
		"https://github.com/hatchet-dev/hatchet/actions/runs/1\n":   false,
	}
	for in, want := range urls {
		if got := runURLRE.MatchString(in); got != want {
			t.Errorf("runURLRE(%q) = %v, want %v", in, got, want)
		}
	}
}

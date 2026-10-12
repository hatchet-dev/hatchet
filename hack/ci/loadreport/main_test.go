package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"
)

const header = "# host: Linux 6.8.0 x86_64\n# cpus: 4\n# columns: time load1 ...\n"

const evilProc = "`](https://evil.example)@octocat"

// synthLog writes n well-formed samples, every one naming evilProc.
func synthLog(n int) string {
	var b strings.Builder
	b.WriteString(header)
	for i := 0; i < n; i++ {
		busy := 50 + float64(i%50)
		fmt.Fprintf(&b, "10:%02d:%02d 3.1 2.0 %.1f 40 10 1.0 %.1f 8000 loadtest=30.0 postgres=20.5 %s=1.0 kworker/u8:1=0.2\n",
			(i*2)/60, (i*2)%60, busy, float64(i%10), evilProc)
	}
	return b.String()
}

func writeFiles(t *testing.T, files map[string]string) map[string]string {
	t.Helper()
	dir := t.TempDir()
	paths := map[string]string{}
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		paths[name] = p
	}
	return paths
}

// outsideCode returns the markdown with fenced blocks and inline code spans
// removed, which is where untrusted text must never appear raw.
func outsideCode(md string) string {
	var out []string
	in := false
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "```") {
			in = !in
			continue
		}
		if !in {
			out = append(out, regexp.MustCompile("`[^`]*`").ReplaceAllString(line, ""))
		}
	}
	return strings.Join(out, "\n")
}

func fencesBalanced(md string) bool {
	n := 0
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "```") {
			n++
		}
	}
	return n%2 == 0
}

func TestParseLogSkipsGarbage(t *testing.T) {
	text := synthLog(3) +
		"garbage line here\n" +
		"10:99:9x 1 1 1 1 1 1 1 7\n" +
		"10:00:00 NaN 1 1 1 1 1 1 7\n" +
		"10:00:00 1 1 Inf 1 1 1 1 7\n" +
		"10:00:00 1 1 1 1 1 1 1\n" +
		"10:00:00 1 1 1 1 1 1 1 7 noequals\n" +
		"10:00:00 1 1 1 1 1 1 1 7 =1\n" +
		"10:00:00 1 1 1 1 1 1 1 7 x=NaN\n" +
		"\n"
	log := parseLog(text)
	if len(log.rows) != 3 || log.malformed != 8 {
		t.Fatalf("rows %d malformed %d", len(log.rows), log.malformed)
	}
	if len(log.header) != 2 {
		t.Errorf("header kept %q", log.header)
	}
	if _, ok := log.rows[0].procs["____url_removed_"]; !ok {
		t.Errorf("evil name not reduced: %v", log.rows[0].procs)
	}
	lines := summaryLines(log)
	if !contains(lines, "malformed samples: 8") || !contains(lines, "samples: 3") {
		t.Errorf("summary %q", lines)
	}
}

func contains(lines []string, s string) bool {
	for _, l := range lines {
		if l == s {
			return true
		}
	}
	return false
}

func TestRunRejectsBadLabelAndMissingLog(t *testing.T) {
	p := writeFiles(t, map[string]string{"cpu.log": synthLog(2)})
	if _, err := run(options{cpu: p["cpu.log"], label: "[link](x)", maxBytes: 60000}); err == nil {
		t.Error("label with markdown accepted")
	}
	if _, err := run(options{cpu: filepath.Join(t.TempDir(), "missing"), label: "x", maxBytes: 60000}); err == nil {
		t.Error("missing cpu log accepted")
	}
}

func TestRunLinkIsBuiltOrOmitted(t *testing.T) {
	p := writeFiles(t, map[string]string{"cpu.log": synthLog(2)})
	bad := [][2]string{
		{"evil.example/hatchet-dev/hatchet", "1"},
		{"../hatchet", "1"},
		{"hatchet-dev/..", "1"},
		{"./hatchet", "1"},
		{"hatchet-dev/hatchet/extra", "1"},
		{"hatchet-dev@evil/hatchet", "1"},
		{"hatchet-dev/hatchet", "1@evil"},
		{"hatchet-dev/hatchet", "-1"},
		{"hatchet-dev/hatchet", ""},
		{"", "1"},
	}
	for _, c := range bad {
		if got := runLink(c[0], c[1]); got != "" {
			t.Errorf("runLink(%q, %q) = %q", c[0], c[1], got)
		}
		out, err := run(options{cpu: p["cpu.log"], label: "x", repository: c[0], runID: c[1], maxBytes: 60000})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "evil") || strings.Contains(out, "Raw samples") {
			t.Errorf("bad run context %v leaked into %q", c, out)
		}
	}
	good := "https://github.com/hatchet-dev/hatchet/actions/runs/1"
	out, err := run(options{cpu: p["cpu.log"], label: "x", repository: "hatchet-dev/hatchet", runID: "1", maxBytes: 60000})
	if err != nil || !strings.HasSuffix(out, "artifacts: "+good+"\n") {
		t.Errorf("good run link missing: %v %q", err, out)
	}
}

func TestAdversarialInputsStayInsideCode(t *testing.T) {
	p := writeFiles(t, map[string]string{
		"cpu.log":    synthLog(50) + "junk\n",
		"pprof.txt":  "top:\n  ghp_abcdefghijklmnop123 AKIAABCDEFGHIJKLMNOP eyJhbGciOi.eyJzdWIi.SflKxw \x1b[31mred\x1b[0m\n  f @octocat https://evil.example/x\n",
		"pgstat.txt": "top statements:\n  12 3 4 5  SELECT 1; ```\n<img onerror=alert(1) src=x>\n~~~\n",
	})
	out, err := run(options{cpu: p["cpu.log"], pprof: p["pprof.txt"], pgstat: p["pgstat.txt"],
		label: "load (optimistic true, race false)", maxBytes: 60000})
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"ghp_", "AKIA", "eyJ", "\x1b", "evil.example", "@octocat", "](", "onerror"} {
		if strings.Contains(outsideCode(out), leak) {
			t.Errorf("%q appears outside code:\n%s", leak, out)
		}
	}
	for _, leak := range []string{"ghp_", "AKIA", "eyJ", "\x1b", "evil.example", "@octocat", "~~~"} {
		if strings.Contains(out, leak) {
			t.Errorf("%q appears in the comment:\n%s", leak, out)
		}
	}
	if !fencesBalanced(out) {
		t.Errorf("fences unbalanced:\n%s", out)
	}
	if strings.Count(out, "<details>") != 2 || strings.Count(out, "<img") != 1 {
		t.Errorf("unexpected html:\n%s", out)
	}
	if !strings.Contains(out, "malformed samples: 1") {
		t.Errorf("junk line not counted:\n%s", out)
	}
}

func TestSizeDropOrder(t *testing.T) {
	big := strings.Repeat(strings.Repeat("p", 300)+"\n", 150)
	p := writeFiles(t, map[string]string{"cpu.log": synthLog(100), "pprof.txt": big, "pgstat.txt": big})
	url := "https://github.com/hatchet-dev/hatchet/actions/runs/1"
	full, err := run(options{cpu: p["cpu.log"], pprof: p["pprof.txt"], pgstat: p["pgstat.txt"], label: "x", repository: "hatchet-dev/hatchet", runID: "1", maxBytes: 200000})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		max                                        int
		pgstat, pprof, processChart, pressureChart bool
	}{
		{200000, true, true, true, true},
		{60000, false, true, true, true},
		{12000, false, false, true, true},
		{3000, false, false, false, true},
		{1500, false, false, false, false},
	}
	for _, c := range cases {
		out, err := run(options{cpu: p["cpu.log"], pprof: p["pprof.txt"], pgstat: p["pgstat.txt"], label: "x", repository: "hatchet-dev/hatchet", runID: "1", maxBytes: c.max})
		if err != nil {
			t.Fatalf("max %d: %v", c.max, err)
		}
		if len(out) > c.max {
			t.Errorf("max %d: got %d bytes", c.max, len(out))
		}
		got := [4]bool{strings.Contains(out, "pg_stat:"), strings.Contains(out, "pprof:"),
			strings.Contains(out, "Top processes"), strings.Contains(out, "Runner pressure")}
		if got != [4]bool{c.pgstat, c.pprof, c.processChart, c.pressureChart} {
			t.Errorf("max %d: sections %v", c.max, got)
		}
		if !fencesBalanced(out) || !strings.HasSuffix(out, "artifacts: "+url+"\n") || !strings.HasPrefix(out, "### Runner CPU") {
			t.Errorf("max %d: malformed:\n%s", c.max, out)
		}
		if c.max < 3000 && !strings.Contains(out, truncatedNote) {
			t.Errorf("max %d: summary not truncated:\n%s", c.max, out)
		}
	}
	if len(full) <= 60000 {
		t.Errorf("fixture too small to exercise dropping: %d", len(full))
	}
	if _, err := run(options{cpu: p["cpu.log"], label: "x", maxBytes: 10}); err == nil {
		t.Error("impossible cap accepted")
	}
}

func TestGoldenCharts(t *testing.T) {
	p := writeFiles(t, map[string]string{"cpu.log": synthLog(100), "pprof.txt": "cpu profiles: 1\n", "pgstat.txt": ""})
	out, err := run(options{cpu: p["cpu.log"], pprof: p["pprof.txt"], pgstat: p["pgstat.txt"],
		label: "load (optimistic true, race false)", repository: "hatchet-dev/hatchet", runID: "1", maxBytes: 60000})
	if err != nil {
		t.Fatal(err)
	}
	axisRE := regexp.MustCompile(`^    x-axis "sample time" \[("\d\d:\d\d:\d\d"(, )?)+\]$`)
	lineRE := regexp.MustCompile(`^    line "[A-Za-z0-9._:/+-]{1,40}" \[(-?\d+\.\d(, )?)+\]$`)
	charts, axes, lines := 0, 0, 0
	for _, l := range strings.Split(out, "\n") {
		switch {
		case l == "```mermaid":
			charts++
		case strings.HasPrefix(l, "    x-axis"):
			axes++
			if !axisRE.MatchString(l) {
				t.Errorf("bad axis %q", l)
			}
			// Mermaid gets unreadable past a few dozen points, so the axis
			// must hold exactly the bucket count; its own quoted title is not
			// a point.
			if n := strings.Count(l, "\"")/2 - 1; n != maxBuckets {
				t.Errorf("axis has %d points", n)
			}
		case strings.HasPrefix(l, "    line"):
			lines++
			if !lineRE.MatchString(l) || strings.Count(l, ",") != maxBuckets-1 {
				t.Errorf("bad line %q", l)
			}
		case strings.HasPrefix(l, "    title"), strings.HasPrefix(l, "    y-axis"), l == "xychart-beta":
		case strings.HasPrefix(l, "    "):
			t.Errorf("unexpected chart line %q", l)
		}
	}
	if charts != 2 || axes != 2 || lines != 2+4 {
		t.Errorf("charts %d axes %d lines %d", charts, axes, lines)
	}
	want := []string{
		"### Runner CPU during the failed `load (optimistic true, race false)` run\n",
		"samples: 100\nmalformed samples: 0\nbusy%: mean 74.5, max 99.0, samples over 90% busy: 18 (18% of the run)\n",
		"steal%: mean 4.5, max 9.0, samples over 5% steal: 40 (40% of the run)\niowait%: mean 1.0\nload1: max 3.1\n",
		"top processes by mean share of all CPUs:\n  loadtest                  30.0%\n  postgres                  20.5%\n  ____url_removed_           1.0%\n  kworker/u8:1               0.2%\n```\n",
		"    title \"Runner pressure (100 samples)\"\n",
		"    line \"busy\" [50.5, 53.0, 55.5, 58.0, 60.5, 63.0, 65.5, 68.0, 70.5, 73.0, 75.5, 78.0, 80.5, 83.0, 85.5, 88.0, 90.5, ",
		"Process lines, in order: `loadtest`, `postgres`, `____url_removed_`, `kworker/u8:1`\n",
		"<details><summary>pprof: where the load test process spent its CPU</summary>\n\n```\ncpu profiles: 1\n```\n\n</details>\n",
		"<details><summary>pg_stat: where Postgres spent its time</summary>\n\n```\n(no summary captured)\n```\n\n</details>\n",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in:\n%s", w, out)
		}
	}
}

func TestNoSamples(t *testing.T) {
	p := writeFiles(t, map[string]string{"cpu.log": header + "junk\n"})
	out, err := run(options{cpu: p["cpu.log"], label: "x", maxBytes: 60000})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "mermaid") || !strings.Contains(out, "no cpu samples recorded\nmalformed samples: 1\n") || !fencesBalanced(out) {
		t.Errorf("unexpected:\n%s", out)
	}
}

// A credential-shaped process name must not reach any sink, including the
// chart series and the inline process list that bypass sanitizeBlock.
func TestProcessNameSecretsRedactedInEverySink(t *testing.T) {
	p := writeFiles(t, map[string]string{"cpu.log": header +
		"10:00:00 1.0 1.0 50.0 40 10 0.0 0.0 8000 ghp_abcdefghijklmnop123=60 postgresql://u:DemoPgPassword42@h=10\n" +
		"10:00:02 1.0 1.0 50.0 40 10 0.0 0.0 8000 ghp_abcdefghijklmnop123=60 postgresql://u:DemoPgPassword42@h=10\n"})
	out, err := run(options{cpu: p["cpu.log"], label: "x", maxBytes: 60000})
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"ghp_", "abcdefghijklmnop", "DemoPgPassword42"} {
		if strings.Contains(out, leak) {
			t.Errorf("%q leaked:\n%s", leak, out)
		}
	}
	if strings.Count(out, "_redacted_") < 3 {
		t.Errorf("expected the redacted name in the summary, the chart and the list:\n%s", out)
	}
}

// The label is allowlisted but a bare host is still autolinkable, so it is
// rendered inside a code span.
func TestLabelRendersInsideCodeSpan(t *testing.T) {
	p := writeFiles(t, map[string]string{"cpu.log": synthLog(2)})
	out, err := run(options{cpu: p["cpu.log"], label: "www.attacker.example", maxBytes: 60000})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "### Runner CPU during the failed `www.attacker.example` run\n") {
		t.Errorf("heading:\n%s", out)
	}
	if strings.Contains(outsideCode(out), "attacker") {
		t.Errorf("label appears outside code:\n%s", out)
	}
	if n := regexp.MustCompile("`www\\.attacker\\.example`").FindAllString(out, -1); len(n) != 1 || strings.Count(out, "attacker") != 1 {
		t.Errorf("label not confined to one code span:\n%s", out)
	}
}

// The reporter runs late in a job that is already failing, so an input a
// test could make unbounded or unreadable, such as a FIFO, must be rejected
// before it can stall the step or exhaust its memory.
func TestInputBounds(t *testing.T) {
	dir := t.TempDir()
	huge := filepath.Join(dir, "huge.log")
	f, err := os.Create(huge)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(9 << 20); err != nil {
		t.Fatal(err)
	}
	f.Close()
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skip("mkfifo unavailable:", err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(huge, link); err != nil {
		t.Fatal(err)
	}
	good := writeFiles(t, map[string]string{"cpu.log": synthLog(2)})

	for _, path := range []string{huge, fifo, link, dir} {
		start := time.Now()
		if _, err := run(options{cpu: path, label: "x", maxBytes: 60000}); err == nil {
			t.Errorf("%s accepted as -cpu", path)
		}
		if _, err := run(options{cpu: good["cpu.log"], pprof: path, label: "x", maxBytes: 60000}); err == nil {
			t.Errorf("%s accepted as -pprof", path)
		}
		if time.Since(start) > 2*time.Second {
			t.Errorf("%s took %s to reject", path, time.Since(start))
		}
	}

	var b strings.Builder
	b.WriteString(header)
	for i := 0; i < maxSamples+5; i++ {
		b.WriteString("10:00:00 1.0 1.0 50.0 40 10 0.0 0.0 8000 a=1\n")
	}
	many := strings.Repeat(" p=1", maxProcs+1)
	b.WriteString("10:00:00 1.0 1.0 50.0 40 10 0.0 0.0 8000" + many + "\n")
	b.WriteString("10:00:00 1.0 1.0 101.0 40 10 0.0 0.0 8000 a=1\n")
	b.WriteString("10:00:00 1.0 1.0 50.0 40 10 0.0 -1.0 8000 a=1\n")
	b.WriteString("10:00:00 1.0 1.0 50.0 40 10 0.0 0.0 8000 a=100.5\n")
	b.WriteString("10:00:00 -1.0 1.0 50.0 40 10 0.0 0.0 8000 a=1\n")
	log := parseLog(b.String())
	if len(log.rows) != maxSamples || log.dropped != 10 {
		t.Errorf("rows %d dropped %d malformed %d", len(log.rows), log.dropped, log.malformed)
	}
	log = parseLog(header + "10:00:00 1.0 1.0 50.0 40 10 0.0 0.0 8000" + many + "\n" +
		"10:00:00 1.0 1.0 101.0 40 10 0.0 0.0 8000 a=1\n" +
		"10:00:00 1.0 1.0 50.0 40 10 0.0 -1.0 8000 a=1\n" +
		"10:00:00 1.0 1.0 50.0 40 10 0.0 0.0 8000 a=100.5\n" +
		"10:00:00 -1.0 1.0 50.0 40 10 0.0 0.0 8000 a=1\n" +
		"10:00:00 1.0 1.0 100.0 40 10 0.0 0.0 8000 a=100\n")
	if len(log.rows) != 1 || log.malformed != 5 {
		t.Errorf("rows %d malformed %d", len(log.rows), log.malformed)
	}
	if !contains(summaryLines(sampleLog{rows: log.rows, dropped: 3}), "samples dropped past the cap: 3") {
		t.Error("dropped count not reported")
	}
}

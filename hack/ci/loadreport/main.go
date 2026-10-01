// Command loadreport renders the pull request comment the CI load job posts
// when a load test fails. It reads the sample log written by
// hack/ci/cpu-sampler.sh and the plain-text summaries written by
// hack/ci/pprof-sampler.sh and hack/ci/pgstat-sampler.sh, and prints GitHub
// markdown on stdout: a CPU summary, a Mermaid chart of busy and steal over
// time, a chart of the top processes, and the two summaries folded into
// details sections.
//
//	go run ./hack/ci/loadreport -cpu cpu-samples.log \
//	    -pprof pprof/summary.txt -pgstat pgstat/summary.txt \
//	    -label "load (optimistic true, race false)" \
//	    -run-url https://github.com/hatchet-dev/hatchet/actions/runs/1 \
//	    [-max-bytes 60000]
//
// The summaries are optional and may be missing or empty. Everything read is
// untrusted; sanitize.go documents the policy. On any input error a short
// message goes to stderr, the exit code is 1 and nothing is written to stdout.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	maxBuckets   = 40
	topInSummary = 8
	topInChart   = 5
)

type options struct {
	cpu, pprof, pgstat, label, runURL string
	maxBytes                          int
}

type sample struct {
	t                          string
	load1, busy, iowait, steal float64
	procs                      map[string]float64
}

type sampleLog struct {
	header    []string
	rows      []sample
	malformed int
}

type procShare struct {
	name string
	mean float64
}

// block is an optional fenced section; present is false when no path was
// given for it at all.
type block struct {
	present bool
	lines   []string
}

type report struct {
	label, runURL string
	summary       []string
	pressureChart string
	processChart  string
	processNames  []string
	pprof, pgstat block
}

func main() {
	var o options
	flag.StringVar(&o.cpu, "cpu", "", "CPU sample log written by cpu-sampler.sh (required)")
	flag.StringVar(&o.pprof, "pprof", "", "pprof-sampler.sh summarize output (optional)")
	flag.StringVar(&o.pgstat, "pgstat", "", "pgstat-sampler.sh summarize output (optional)")
	flag.StringVar(&o.label, "label", "load test", "matrix label named in the heading")
	flag.StringVar(&o.runURL, "run-url", "", "GitHub Actions run URL linked at the end")
	flag.IntVar(&o.maxBytes, "max-bytes", 60000, "largest comment to emit")
	flag.Parse()

	out, err := run(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "loadreport:", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.WriteString(out); err != nil {
		os.Exit(1)
	}
}

func run(o options) (string, error) {
	if !labelRE.MatchString(o.label) {
		return "", errors.New("label must match " + labelRE.String())
	}
	if o.cpu == "" {
		return "", errors.New("-cpu is required")
	}
	data, err := os.ReadFile(o.cpu)
	if err != nil {
		return "", errors.New("cannot read the cpu sample log")
	}
	pprof, err := readSummary(o.pprof)
	if err != nil {
		return "", errors.New("cannot read the pprof summary")
	}
	pgstat, err := readSummary(o.pgstat)
	if err != nil {
		return "", errors.New("cannot read the pg_stat summary")
	}

	log := parseLog(string(data))
	r := report{label: o.label, summary: summaryLines(log), pprof: pprof, pgstat: pgstat}
	if runURLRE.MatchString(o.runURL) {
		r.runURL = o.runURL
	}
	if len(log.rows) > 0 {
		r.pressureChart, r.processChart, r.processNames = charts(log.rows)
	}
	return assemble(r, o.maxBytes)
}

// readSummary treats a missing file like an empty one: the summary steps run
// with if: always() but can still fail before writing anything.
func readSummary(path string) (block, error) {
	if path == "" {
		return block{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return block{}, err
	}
	text := string(data)
	if strings.TrimSpace(text) == "" {
		text = "(no summary captured)"
	}
	return block{present: true, lines: sanitizeBlock(text)}, nil
}

func parseLog(text string) sampleLog {
	var log sampleLog
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.TrimSpace(line) == "":
		case strings.HasPrefix(line, "#"):
			if !strings.HasPrefix(line, "# columns") {
				log.header = append(log.header, line)
			}
		default:
			if s, ok := parseSample(line); ok {
				log.rows = append(log.rows, s)
			} else {
				log.malformed++
			}
		}
	}
	return log
}

// parseSample accepts only a line whose shape matches the sampler's columns
// exactly: time load1 load5 busy user system iowait steal mem [name=pct ...].
func parseSample(line string) (sample, bool) {
	f := strings.Fields(line)
	if len(f) < 9 || !timeRE.MatchString(f[0]) {
		return sample{}, false
	}
	var nums [8]float64
	for i := range nums {
		v, ok := finite(f[i+1])
		if !ok {
			return sample{}, false
		}
		nums[i] = v
	}
	s := sample{t: f[0], load1: nums[0], busy: nums[2], iowait: nums[5], steal: nums[6], procs: map[string]float64{}}
	for _, tok := range f[9:] {
		eq := strings.LastIndexByte(tok, '=')
		if eq <= 0 {
			return sample{}, false
		}
		v, ok := finite(tok[eq+1:])
		if !ok {
			return sample{}, false
		}
		s.procs[safeName(tok[:eq])] += v
	}
	return s, true
}

func finite(s string) (float64, bool) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// topProcs ranks commands by mean share over all samples, a sample where the
// command is absent counting as zero, as the shell summary does.
func topProcs(rows []sample, limit int) []procShare {
	sum := map[string]float64{}
	for _, r := range rows {
		for name, v := range r.procs {
			sum[name] += v
		}
	}
	top := make([]procShare, 0, len(sum))
	for name, v := range sum {
		top = append(top, procShare{name, v / float64(len(rows))})
	}
	sort.Slice(top, func(i, j int) bool {
		if top[i].mean != top[j].mean {
			return top[i].mean > top[j].mean
		}
		return top[i].name < top[j].name
	})
	if len(top) > limit {
		top = top[:limit]
	}
	return top
}

// summaryLines reproduces the shell summarize output and runs all of it
// through sanitizeBlock, so the header lines copied from the log are covered
// by the same rules as the other fenced text.
func summaryLines(log sampleLog) []string {
	var b strings.Builder
	for _, h := range log.header {
		b.WriteString(h + "\n")
	}
	n := len(log.rows)
	if n == 0 {
		b.WriteString("no cpu samples recorded\n")
		fmt.Fprintf(&b, "malformed samples: %d\n", log.malformed)
		return sanitizeBlock(b.String())
	}
	var busy, steal, iowait, busyMax, stealMax, load1Max float64
	var saturated, stolen int
	for _, r := range log.rows {
		busy += r.busy
		steal += r.steal
		iowait += r.iowait
		busyMax = math.Max(busyMax, r.busy)
		stealMax = math.Max(stealMax, r.steal)
		load1Max = math.Max(load1Max, r.load1)
		if r.busy > 90 {
			saturated++
		}
		if r.steal > 5 {
			stolen++
		}
	}
	fn := float64(n)
	fmt.Fprintf(&b, "samples: %d\n", n)
	fmt.Fprintf(&b, "malformed samples: %d\n", log.malformed)
	fmt.Fprintf(&b, "busy%%: mean %s, max %s, samples over 90%% busy: %d (%.0f%% of the run)\n",
		num(busy/fn), num(busyMax), saturated, float64(saturated)*100/fn)
	fmt.Fprintf(&b, "steal%%: mean %s, max %s, samples over 5%% steal: %d (%.0f%% of the run)\n",
		num(steal/fn), num(stealMax), stolen, float64(stolen)*100/fn)
	fmt.Fprintf(&b, "iowait%%: mean %s\n", num(iowait/fn))
	fmt.Fprintf(&b, "load1: max %s\n", num(load1Max))
	b.WriteString("top processes by mean share of all CPUs:\n")
	for _, p := range topProcs(log.rows, topInSummary) {
		fmt.Fprintf(&b, "  %-24s %5s%%\n", p.name, num(p.mean))
	}
	return sanitizeBlock(b.String())
}

// charts averages the samples into at most maxBuckets buckets, since Mermaid
// charts get unreadable past a few dozen points, and renders both charts.
func charts(rows []sample) (pressure, processes string, names []string) {
	n := len(rows)
	buckets := n
	if buckets > maxBuckets {
		buckets = maxBuckets
	}
	per := float64(n) / float64(buckets)
	labels := make([]string, buckets)
	lo := make([]int, buckets)
	hi := make([]int, buckets)
	for b := range labels {
		lo[b], hi[b] = int(float64(b)*per), int(float64(b+1)*per)-1
		if hi[b] < lo[b] {
			hi[b] = lo[b]
		}
		if hi[b] > n-1 {
			hi[b] = n - 1
		}
		labels[b] = rows[lo[b]].t
	}
	mean := func(get func(sample) float64) []float64 {
		out := make([]float64, buckets)
		for b := range out {
			var sum float64
			for i := lo[b]; i <= hi[b]; i++ {
				sum += get(rows[i])
			}
			out[b] = sum / float64(hi[b]-lo[b]+1)
		}
		return out
	}

	busy := mean(func(s sample) float64 { return s.busy })
	steal := mean(func(s sample) float64 { return s.steal })
	pressure = chart(fmt.Sprintf("Runner pressure (%d samples)", n), labels,
		[]series{{"busy", busy}, {"steal", steal}})

	var top []series
	for _, p := range topProcs(rows, topInChart) {
		name := p.name
		top = append(top, series{name, mean(func(s sample) float64 { return s.procs[name] })})
		names = append(names, name)
	}
	if len(top) > 0 {
		processes = chart("Top processes, share of all CPUs", labels, top)
	}
	return pressure, processes, names
}

type series struct {
	name   string
	values []float64
}

func chart(title string, labels []string, lines []series) string {
	var b strings.Builder
	b.WriteString("```mermaid\nxychart-beta\n")
	fmt.Fprintf(&b, "    title %q\n", mermaidString(title))
	quoted := make([]string, len(labels))
	for i, l := range labels {
		quoted[i] = `"` + mermaidString(l) + `"`
	}
	fmt.Fprintf(&b, "    x-axis \"sample time\" [%s]\n", strings.Join(quoted, ", "))
	b.WriteString("    y-axis \"percent of all CPUs\" 0 --> 100\n")
	for _, s := range lines {
		nums := make([]string, len(s.values))
		for i, v := range s.values {
			nums[i] = num(v)
		}
		fmt.Fprintf(&b, "    line \"%s\" [%s]\n", mermaidString(s.name), strings.Join(nums, ", "))
	}
	b.WriteString("```\n")
	return b.String()
}

// fence closes every block it opens, truncating to limit lines first.
func fence(lines []string, limit int) string {
	var b strings.Builder
	b.WriteString("```\n")
	for i, l := range lines {
		if i == limit {
			b.WriteString(truncatedNote + "\n")
			break
		}
		b.WriteString(l + "\n")
	}
	b.WriteString("```\n")
	return b.String()
}

func details(summary string, lines []string, limit int) string {
	return "<details><summary>" + summary + "</summary>\n\n" + fence(lines, limit) + "\n</details>\n"
}

type layout struct {
	pgstat, pprof, processChart, pressureChart bool
	summaryLimit                               int
}

// assemble drops sections from the end until the comment fits, then shrinks
// the summary block, so the output is always well-formed markdown.
func assemble(r report, maxBytes int) (string, error) {
	l := layout{pgstat: true, pprof: true, processChart: true, pressureChart: true, summaryLimit: blockMaxLines + 1}
	for {
		out := render(r, l)
		if len(out) <= maxBytes {
			return out, nil
		}
		switch {
		case l.pgstat:
			l.pgstat = false
		case l.pprof:
			l.pprof = false
		case l.processChart:
			l.processChart = false
		case l.summaryLimit > 8:
			l.summaryLimit /= 2
		case l.pressureChart:
			l.pressureChart = false
		case l.summaryLimit > 0:
			l.summaryLimit = 0
		default:
			return "", errors.New("comment does not fit in -max-bytes")
		}
	}
}

func render(r report, l layout) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### Runner CPU during the failed %s run\n\n", r.label)
	b.WriteString("Busy is 100 minus idle across all CPUs: near 100 means the runner was saturated. " +
		"Steal is CPU the host took away from the runner: above a few percent means it was oversubscribed. " +
		"Either way the latency gate measured the runner, not the scheduler.\n\n")
	b.WriteString(fence(r.summary, l.summaryLimit))
	if l.pressureChart && r.pressureChart != "" {
		b.WriteString("\n" + r.pressureChart)
	}
	if l.processChart && r.processChart != "" {
		b.WriteString("\n" + r.processChart)
		quoted := make([]string, len(r.processNames))
		for i, n := range r.processNames {
			quoted[i] = "`" + n + "`"
		}
		b.WriteString("\nProcess lines, in order: " + strings.Join(quoted, ", ") + "\n")
	}
	if l.pprof && r.pprof.present {
		b.WriteString("\n" + details("pprof: where the load test process spent its CPU", r.pprof.lines, blockMaxLines+1))
	}
	if l.pgstat && r.pgstat.present {
		b.WriteString("\n" + details("pg_stat: where Postgres spent its time", r.pgstat.lines, blockMaxLines+1))
	}
	if r.runURL != "" {
		b.WriteString("\nRaw samples are in the run's `load-samples-*` artifacts: " + r.runURL + "\n")
	}
	return b.String()
}

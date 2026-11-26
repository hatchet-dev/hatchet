package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// result builds a valid result for a registry track whose restarts lost the
// given seconds each, using one tenant per restart.
func result(track string, lost ...float64) Result {
	t, ok := trackByName(track)
	if !ok {
		panic("unknown track " + track)
	}
	r := Result{
		Track:     t,
		Steady:    Counts{Offered: 1000},
		Timeline:  make([]float64, timelineBuckets),
		PeakConns: t.MaxConns,
	}
	const offered = 1_000_000
	for _, l := range lost {
		bad := int(math.Round(l / t.WindowS * offered))
		r.Restarts = append(r.Restarts, Restart{DownS: 2, Tenants: []Counts{{Offered: offered, Bad: bad}}})
	}
	return r
}

// spread returns five restarts centred on c, w apart.
func spread(c, w float64) []float64 {
	return []float64{c - 2*w, c - w, c, c + w, c + 2*w}
}

func TestVerdictRules(t *testing.T) {
	const track = "mt-skewed"
	tr, _ := trackByName(track)
	f := tr.NoiseFloorS
	c := 10*f + 10
	// Five restarts spread over a range of f/2, so a shift of more than f/2
	// separates the ranges.
	base := spread(c, f/8)

	tests := []struct {
		name        string
		base, head  []float64
		want        verdict
		fewRestarts bool
	}{
		{"disjoint and beyond the floor improves", base, spread(c-2*f, f/8), improved, false},
		{"disjoint and beyond the floor regresses", base, spread(c+2*f, f/8), regressed, false},
		{"disjoint within the floor is no change", base, spread(c-0.75*f, f/8), noChange, false},
		{"regression within the floor is no change", base, spread(c+0.75*f, f/8), noChange, false},
		{"overlapping ranges are no change even past the floor", base, []float64{c - 3*f, c - 3*f, c - 3*f, c - 3*f, c}, noChange, false},
		{"identical runs are no change", base, base, noChange, false},
		{"too few base restarts is no change", base[:minRestarts-1], spread(c+5*f, f/8), noChange, true},
		{"too few head restarts is no change", base, spread(c-5*f, f/8)[:minRestarts-1], noChange, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmps := compareResults([]Result{result(track, tt.base...)}, []Result{result(track, tt.head...)}, nil)
			if len(cmps) != 1 {
				t.Fatalf("got %d tracks, want 1", len(cmps))
			}
			if got := cmps[0]; got.verdict != tt.want || got.fewRestarts != tt.fewRestarts {
				t.Errorf("got %v (few restarts %v), want %v (few restarts %v)", got.verdict, got.fewRestarts, tt.want, tt.fewRestarts)
			}
		})
	}
}

func TestInvalid(t *testing.T) {
	const track = "mt-many-equal"
	ok := func() Result { return result(track, 1, 1, 1, 1, 1) }
	with := func(edit func(*Result)) Result {
		r := ok()
		edit(&r)
		return r
	}
	otherTrack := result("mt-skewed", 1, 1, 1, 1, 1)

	tests := []struct {
		name       string
		base, head []Result
		side       string
		reason     string
	}{
		{"missing on head", []Result{ok()}, []Result{otherTrack}, "head", "no result for this track"},
		{"missing on base", []Result{otherTrack}, []Result{ok()}, "base", "no result for this track"},
		{"twice in a file", []Result{ok()}, []Result{ok(), ok()}, "head", "more than one result"},
		{"stale track settings", []Result{with(func(r *Result) { r.Track.PeriodS += 1 })}, []Result{ok()}, "base", "different track settings"},
		{"lost trigger", []Result{ok()}, []Result{with(func(r *Result) { r.Lost = 3 })}, "head", "3 accepted trigger(s) never ran"},
		{"duplicate run", []Result{ok()}, []Result{with(func(r *Result) { r.Duplicates = 2 })}, "head", "2 trigger(s) ran more than once"},
		{"max_runs exceeded", []Result{ok()}, []Result{with(func(r *Result) { r.MaxRunsViolations = 1 })}, "head", "exceeded max_runs"},
		{"steady state unhealthy", []Result{ok()}, []Result{with(func(r *Result) { r.Steady.Bad = 20 })}, "head", "outside restart windows were bad"},
		{"nothing measured outside restarts", []Result{ok()}, []Result{with(func(r *Result) { r.Steady = Counts{} })}, "head", "no operations measured outside restart windows"},
		{"too many connections", []Result{with(func(r *Result) { r.PeakConns = r.Track.MaxConns + 1 })}, []Result{ok()}, "base", "exceeds DATABASE_MAX_CONNS"},
		{"no restarts", []Result{ok()}, []Result{with(func(r *Result) { r.Restarts = nil })}, "head", "no restarts recorded"},
		{"starved tenant", []Result{ok()}, []Result{with(func(r *Result) { r.Restarts[2].Tenants[0] = Counts{} })}, "head", "a tenant was offered no operations"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := compareFiles(t, tt.base, tt.head)
			if !strings.HasPrefix(out, "## Restart harness: INVALID\n") {
				t.Errorf("heading is not INVALID:\n%s", out)
			}
			if want := "- " + track + ", " + tt.side + ": "; !strings.Contains(out, want) || !strings.Contains(out, tt.reason) {
				t.Errorf("output lacks %q with %q:\n%s", want, tt.reason, out)
			}
			if !strings.HasSuffix(out, "\nSCORE INVALID\n") {
				t.Errorf("last line is not SCORE INVALID:\n%s", out)
			}
		})
	}
}

func TestOverallPrecedence(t *testing.T) {
	tests := []struct {
		tracks []verdict
		want   verdict
	}{
		{nil, noChange},
		{[]verdict{noChange, noChange}, noChange},
		{[]verdict{noChange, improved}, improved},
		{[]verdict{improved, regressed, noChange}, regressed},
		{[]verdict{improved, regressed, invalid}, invalid},
	}
	for _, tt := range tests {
		var cmps []trackCmp
		for _, v := range tt.tracks {
			cmps = append(cmps, trackCmp{verdict: v})
		}
		if got := overall(cmps); got != tt.want {
			t.Errorf("overall(%v) = %v, want %v", tt.tracks, got, tt.want)
		}
	}
}

func TestMultiTrackReport(t *testing.T) {
	base := []Result{
		result("mt-skewed", 9, 9, 9, 9, 9),
		result("single-tenant-default", 1, 1, 1, 1, 1),
		result("mt-many-equal", 4, 4, 4, 4, 4),
	}
	head := []Result{
		result("mt-many-equal", 4, 4, 4, 4, 4),
		result("single-tenant-default", 1, 2, 3, 4, 5),
		result("mt-skewed", 2, 2, 2, 2, 2),
	}
	out := compareFiles(t, base, head)

	if !strings.HasPrefix(out, "## Restart harness: IMPROVED\n") {
		t.Errorf("heading is not IMPROVED:\n%s", out)
	}
	// Rows follow registry order whatever the order in the files.
	var rows []string
	for _, line := range strings.Split(out, "\n") {
		for _, tr := range tracks {
			if strings.HasPrefix(line, "| "+tr.Name+" | ") && strings.HasSuffix(line, " |") && strings.Count(line, "|") == 7 {
				rows = append(rows, tr.Name)
			}
		}
	}
	if want := []string{"single-tenant-default", "mt-many-equal", "mt-skewed"}; strings.Join(rows, ",") != strings.Join(want, ",") {
		t.Errorf("summary rows %v, want %v", rows, want)
	}
	for _, want := range []string{
		"| mt-skewed | 9.00 (9.00–9.00) | 2.00 (2.00–2.00) | -7.00 |",
		"| total | 14.00 | 9.00 | -5.00 | | IMPROVED |",
		"    x-axis \"seconds since SIGTERM\" -10 --> 29\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if got := strings.Count(out, "```mermaid\nxychart-beta\n"); got != 3 {
		t.Errorf("got %d charts, want 3", got)
	}
	if !strings.HasSuffix(out, "\nSCORE 9.000\n") {
		t.Errorf("last line is not SCORE 9.000:\n%s", out)
	}
}

// A track that was asked for and has no result on either side must show up
// as INVALID, not drop out of the report.
func TestExpectedTrackWithoutResults(t *testing.T) {
	dir := t.TempDir()
	one := marshal(t, result("mt-many-equal", 4, 4, 4, 4, 4))
	base, head := writeFile(t, dir, "base.json", one), writeFile(t, dir, "head.json", one)

	var w bytes.Buffer
	if err := compareCmd([]string{"--tracks", "mt-many-equal,mt-skewed", base, head}, &w); err != nil {
		t.Fatal(err)
	}
	out := w.String()
	if !strings.Contains(out, "- mt-skewed, base: no result for this track") || !strings.HasSuffix(out, "\nSCORE INVALID\n") {
		t.Errorf("missing expected track is not INVALID:\n%s", out)
	}

	if err := compareCmd([]string{"--tracks", "nope", base, head}, &bytes.Buffer{}); err == nil {
		t.Error("an unknown expected track was accepted")
	}
}

func TestFewRestartsStillScores(t *testing.T) {
	out := compareFiles(t,
		[]Result{result("mt-skewed", 3, 3, 3)},
		[]Result{result("mt-skewed", 1.25, 1.25, 1.25)})
	if !strings.Contains(out, "Note: mt-skewed has fewer than 5 restarts") {
		t.Errorf("output lacks the few-restarts note:\n%s", out)
	}
	if !strings.HasSuffix(out, "\nSCORE 1.250\n") {
		t.Errorf("last line is not SCORE 1.250:\n%s", out)
	}
}

// A results file is produced by pull request code in CI, so anything in it
// may be hostile.
func TestHostileInputIsRejected(t *testing.T) {
	const hostile = "@octocat [x](https://evil.example) <img src=x>"
	valid := marshal(t, result("mt-skewed", 1, 1, 1, 1, 1))
	edit := func(f func(*Result)) string {
		r := result("mt-skewed", 1, 1, 1, 1, 1)
		f(&r)
		return marshal(t, r)
	}
	tests := map[string]string{
		"unknown track name":   edit(func(r *Result) { r.Track.Name = hostile }),
		"empty file":           "",
		"not JSON":             hostile,
		"trailing garbage":     valid + hostile,
		"string for a number":  strings.Replace(valid, `"lost":0`, `"lost":"`+hostile+`"`, 1),
		"NaN":                  strings.Replace(valid, `"down_s":2`, `"down_s":NaN`, 1),
		"float overflow":       strings.Replace(valid, `"down_s":2`, `"down_s":1e999`, 1),
		"int overflow":         strings.Replace(valid, `"lost":0`, `"lost":99999999999999999999`, 1),
		"down_s too large":     edit(func(r *Result) { r.Restarts[0].DownS = maxDownS + 1 }),
		"negative error count": edit(func(r *Result) { r.Errors.Timeout = -1 }),
		"negative offered":     edit(func(r *Result) { r.Steady = Counts{Offered: -5, Bad: -10} }),
		"more bad than offered": edit(func(r *Result) {
			r.Restarts[0].Tenants[0] = Counts{Offered: 1, Bad: 1_000_000}
		}),
		"timeline too long":             edit(func(r *Result) { r.Timeline = make([]float64, timelineBuckets+1) }),
		"timeline missing":              edit(func(r *Result) { r.Timeline = nil }),
		"timeline percentage above 100": edit(func(r *Result) { r.Timeline[3] = 1e300 }),
		"too many restarts": edit(func(r *Result) {
			r.Restarts = make([]Restart, maxRestarts+1)
		}),
		"too many tenants": edit(func(r *Result) {
			r.Restarts[0].Tenants = make([]Counts, maxTenants+1)
		}),
		"too many results": strings.Repeat(valid, maxResults+1),
	}
	for name, head := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			var w bytes.Buffer
			err := compareCmd([]string{writeFile(t, dir, "base.json", valid), writeFile(t, dir, "head.json", head)}, &w)
			if err == nil {
				t.Fatalf("no error; output:\n%s", w.String())
			}
			if w.Len() != 0 {
				t.Errorf("wrote output despite the error:\n%s", w.String())
			}
			for _, bad := range []string{"octocat", "evil", "img"} {
				if strings.Contains(err.Error(), bad) {
					t.Errorf("error echoes input: %v", err)
				}
			}
		})
	}
}

func TestOnlyRegularFilesAreRead(t *testing.T) {
	dir := t.TempDir()
	target := writeFile(t, dir, "real.json", marshal(t, result("mt-skewed", 1, 1, 1, 1, 1)))
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{"symlink": link, "directory": dir} {
		if err := compareCmd([]string{target, path}, &bytes.Buffer{}); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	big := writeFile(t, dir, "big.json", strings.Repeat(" ", maxInputBytes+1))
	if err := compareCmd([]string{target, big}, &bytes.Buffer{}); err == nil {
		t.Error("oversized file accepted")
	}
}

// Fields compare does not know, and stale settings with absurd values, parse
// but must never reach the output.
func TestOutputHoldsNoInputText(t *testing.T) {
	r := result("mt-skewed", 1, 1, 1, 1, 1)
	r.Track.WindowS = math.MaxFloat64
	r.PeakConns = math.MaxInt64
	stale := strings.Replace(marshal(t, r), `{"track":{`,
		`{"note":"@octocat https://evil.example","track":{"extra":"<script>","name":"mt-skewed",`, 1)
	dir := t.TempDir()
	base := writeFile(t, dir, "base.json", stale)
	head := writeFile(t, dir, "head.json", marshal(t, result("mt-skewed", 1, 1, 1, 1, 1)))
	var w bytes.Buffer
	if err := compareCmd([]string{base, head}, &w); err != nil {
		t.Fatal(err)
	}
	out := w.String()
	for _, bad := range []string{"@", "http", "evil", "<", "](", "NaN", "Inf"} {
		if strings.Contains(out, bad) {
			t.Errorf("output contains %q:\n%s", bad, out)
		}
	}
	if !strings.HasSuffix(out, "\nSCORE INVALID\n") {
		t.Errorf("last line is not SCORE INVALID:\n%s", out)
	}
}

func compareFiles(t *testing.T, base, head []Result) string {
	t.Helper()
	dir := t.TempDir()
	var w bytes.Buffer
	err := compareCmd([]string{
		writeFile(t, dir, "base.json", marshal(t, base...)),
		writeFile(t, dir, "head.json", marshal(t, head...)),
	}, &w)
	if err != nil {
		t.Fatal(err)
	}
	return w.String()
}

// marshal concatenates results the way a results file does.
func marshal(t *testing.T, rs ...Result) string {
	t.Helper()
	var b strings.Builder
	for _, r := range rs {
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(data)
	}
	return b.String()
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

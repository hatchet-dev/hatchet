package fairpool

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// queryName is the sqlc query name when the statement carries one, otherwise a
// short space-free sketch of the SQL. Counts stay keyed by this so a bucket
// cannot grow one entry per literal.
func queryName(sql string) string {
	line, _, _ := strings.Cut(sql, "\n")
	line = strings.TrimSpace(line)
	const prefix = "-- name:"
	if rest, ok := strings.CutPrefix(line, prefix); ok {
		rest = strings.TrimSpace(rest)
		name, _, _ := strings.Cut(rest, " ")
		if name != "" {
			return name
		}
	}

	sketch := strings.Join(strings.Fields(sql), "_")
	if len(sketch) > 60 {
		sketch = sketch[:60]
	}
	if sketch == "" {
		return "query"
	}

	return sketch
}

func batchLabel(b *pgx.Batch) string {
	if b == nil || len(b.QueuedQueries) == 0 || len(b.QueuedQueries) > 1 {
		return "batch"
	}

	return queryName(b.QueuedQueries[0].SQL)
}

func formatQueryCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}

	type pair struct {
		name string
		n    int
	}
	pairs := make([]pair, 0, len(counts))
	for name, n := range counts {
		if n <= 0 {
			continue
		}
		pairs = append(pairs, pair{name: name, n: n})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].n != pairs[j].n {
			return pairs[i].n > pairs[j].n
		}
		return pairs[i].name < pairs[j].name
	})

	var b strings.Builder
	for i, p := range pairs {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%s=%d", p.name, p.n)
	}

	return b.String()
}

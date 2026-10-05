//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sqlcParam = regexp.MustCompile(`@(\w+)`)

var sqlcFuncParam = regexp.MustCompile(`sqlc\.n?arg\('(\w+)'\)`)

// namedQuery returns a query from streams.sql with its sqlc parameters made
// positional, so a plan test always explains the query that actually ships.
func namedQuery(t *testing.T, name string, args map[string]any) (string, []any) {
	t.Helper()

	src, err := os.ReadFile("sqlcv1/streams.sql")
	require.NoError(t, err)

	start := strings.Index(string(src), "-- name: "+name+" ")
	require.GreaterOrEqual(t, start, 0, "query %s not found", name)

	body := string(src[start:])
	if next := strings.Index(body[1:], "-- name: "); next >= 0 {
		body = body[:next+1]
	}

	// parameters named in comments mustn't take a position
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			lines = append(lines, line)
		}
	}

	body = sqlcFuncParam.ReplaceAllString(strings.Join(lines, "\n"), "@$1")

	positions := map[string]int{}
	var ordered []any

	body = sqlcParam.ReplaceAllStringFunc(body, func(m string) string {
		name := m[1:]

		if _, ok := positions[name]; !ok {
			v, ok := args[name]
			require.True(t, ok, "no value for @%s", name)
			ordered = append(ordered, v)
			positions[name] = len(ordered)
		}

		return "$" + strconv.Itoa(positions[name])
	})

	return body, ordered
}

type explainNode struct {
	NodeType         string        `json:"Node Type"`
	RelationName     string        `json:"Relation Name"`
	SharedHitBlocks  int           `json:"Shared Hit Blocks"`
	SharedReadBlocks int           `json:"Shared Read Blocks"`
	Plans            []explainNode `json:"Plans"`
}

func explain(t *testing.T, pool *pgxpool.Pool, analyze bool, query string, args []any) explainNode {
	t.Helper()

	opts := "FORMAT JSON"
	if analyze {
		opts = "ANALYZE, BUFFERS, FORMAT JSON"
	}

	var raw []byte
	require.NoError(t, pool.QueryRow(context.Background(), "EXPLAIN ("+opts+") "+query, args...).Scan(&raw))

	var out []struct {
		Plan explainNode `json:"Plan"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))

	return out[0].Plan
}

func walk(n explainNode, visit func(explainNode)) {
	visit(n)
	for _, c := range n.Plans {
		walk(c, visit)
	}
}

// Plans for the hot-path queries, on a database of their own so other tests'
// rows can't change them.
func TestStreamsQueryPlans(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	// Catching up or tailing a large topic must read about a page's worth of
	// index, not the whole topic: the primary key puts id ahead of inserted_at
	// so each hourly partition can seek straight to the cursor.
	t.Run("a page read touches a page, not the topic", func(t *testing.T) {
		tenantId := uuid.New()
		const total = 30000

		_, err := pool.Exec(ctx, `SELECT create_v1_hourly_range_partition('v1_stream_message', NOW() - make_interval(hours => h)) FROM generate_series(1, 6) AS h`)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO v1_stream_message (id, tenant_id, topic, payload, producer_id, producer_seq, inserted_at)
			SELECT g, $1, 't', 'x', 'p', g, NOW() - make_interval(secs => ($2 - g) * (6 * 3600.0 / $2))
			FROM generate_series(1, $2) g`, tenantId, total)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `ANALYZE v1_stream_message`)
		require.NoError(t, err)

		var maxID int64
		require.NoError(t, pool.QueryRow(ctx, `SELECT max(id) FROM v1_stream_message WHERE tenant_id = $1`, tenantId).Scan(&maxID))

		for label, afterID := range map[string]int64{"catch-up": 0, "tail": maxID - 5} {
			query, args := namedQuery(t, "ListStreamMessagesAfterCursor", map[string]any{
				"tenantId": tenantId, "namespace": "", "topic": "t", "afterId": afterID,
				"retainedSince": time.Now().Add(-720 * time.Hour), "limit": int32(500), "maxBytes": int64(MaxListStreamMessagesBytes),
			})

			plan := explain(t, pool, true, query, args)

			walk(plan, func(n explainNode) {
				assert.NotEqual(t, "Hash Join", n.NodeType, "%s: joined against the whole topic", label)
			})

			// the whole topic is well over 700 blocks; a page is a few dozen. Small
			// partitions (like the newest hour) may be scanned outright, which is fine.
			assert.Less(t, plan.SharedHitBlocks+plan.SharedReadBlocks, 200, "%s read %d blocks", label, plan.SharedHitBlocks+plan.SharedReadBlocks)
		}
	})

	// Every publish looks up its producer's latest cursor row. It must only
	// plan (and so lock) the cursor partitions within retention, however many
	// older ones exist.
	t.Run("a publish plans only retained cursor partitions", func(t *testing.T) {
		_, err := pool.Exec(ctx, `SELECT create_v1_range_partition('v1_stream_producer_cursor', (NOW() AT TIME ZONE 'UTC')::date - d, 80) FROM generate_series(1, 120) AS d`)
		require.NoError(t, err)

		query, args := namedQuery(t, "InsertOrderedStreamMessage", map[string]any{
			"tenantId": uuid.New(), "namespace": "", "topic": "t", "producerId": "p", "minBucket": streamProducerCursorMinBucket(time.Now()),
			"producerSeq": int64(1), "expectedPrevSeq": int64(0), "payload": []byte("m"), "messageOffset": int64(1),
			"payloadId": nil, "payloadInsertedAt": nil,
		})

		partitions := map[string]struct{}{}
		walk(explain(t, pool, false, query, args), func(n explainNode) {
			if strings.HasPrefix(n.RelationName, "v1_stream_producer_cursor_") {
				partitions[n.RelationName] = struct{}{}
			}
		})

		// within retention: the last few days, plus today and tomorrow
		assert.LessOrEqual(t, len(partitions), int(streamProducerCursorRetention/(24*time.Hour))+2, "planned %v", partitions)
	})
}

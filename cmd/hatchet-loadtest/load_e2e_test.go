//go:build load

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/loadtest/eventkeys"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/random"
	"github.com/hatchet-dev/hatchet/pkg/testing/harness"
)

func TestMain(m *testing.M) {
	harness.RunTestWithEngine(m)
}

func TestLoadCLI(t *testing.T) {
	// We're using LoadTestConfig directly instead of an args struct

	l = logger.NewStdErr(
		&shared.LoggerConfigFile{
			Level:  "warn",
			Format: "console",
		},
		"loadtest",
	)

	avgThreshold := 300 * time.Millisecond
	if v := os.Getenv("HATCHET_LOADTEST_AVERAGE_DURATION_THRESHOLD"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			avgThreshold = parsed
		} else {
			t.Fatalf("invalid HATCHET_LOADTEST_AVERAGE_DURATION_THRESHOLD=%q: %v", v, err)
		}
	}

	startupSleep := 15 * time.Second
	if v := os.Getenv("HATCHET_LOADTEST_STARTUP_SLEEP"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			startupSleep = parsed
		} else {
			t.Fatalf("invalid HATCHET_LOADTEST_STARTUP_SLEEP=%q: %v", v, err)
		}
	}

	tests := []struct {
		name    string
		config  LoadTestConfig
		wantErr bool
		// dagOperator runs the workflow on the DAG operator. The operator is a
		// tenant entitlement that registration applies to every workflow with
		// more than one task, and this table has exactly one such workflow, so
		// enabling it for the tenant changes only the entry that sets this.
		dagOperator bool
	}{
		{
			name: "test simple workflow",
			config: LoadTestConfig{
				Duration:                 240 * time.Second,
				Events:                   10,
				Delay:                    0 * time.Second,
				Wait:                     60 * time.Second,
				Concurrency:              0,
				Slots:                    100,
				FailureRate:              0.0,
				PayloadSize:              "0kb",
				EventFanout:              1,
				DagSteps:                 1,
				RlKeys:                   0,
				RlLimit:                  0,
				RlDurationUnit:           "",
				AverageDurationThreshold: avgThreshold,
			},
		},
		{
			name:        "test with DAG",
			dagOperator: true,
			config: LoadTestConfig{
				Duration:                 240 * time.Second,
				Events:                   10,
				Delay:                    0 * time.Second,
				Wait:                     60 * time.Second,
				Concurrency:              0,
				Slots:                    100,
				FailureRate:              0.0,
				PayloadSize:              "0kb",
				EventFanout:              1,
				DagSteps:                 2,
				RlKeys:                   0,
				RlLimit:                  0,
				RlDurationUnit:           "",
				AverageDurationThreshold: avgThreshold,
			},
		},
		{
			name: "test with event fanout",
			config: LoadTestConfig{
				Duration:                 240 * time.Second,
				Events:                   10,
				Delay:                    0 * time.Second,
				Wait:                     60 * time.Second,
				Concurrency:              0,
				Slots:                    100,
				FailureRate:              0.0,
				PayloadSize:              "0kb",
				EventFanout:              2,
				DagSteps:                 1,
				RlKeys:                   0,
				RlLimit:                  0,
				RlDurationUnit:           "",
				AverageDurationThreshold: avgThreshold,
			},
		},
		{
			name: "test with global concurrency key",
			config: LoadTestConfig{
				Duration:                 240 * time.Second,
				Events:                   10,
				Delay:                    0 * time.Second,
				Wait:                     60 * time.Second,
				Concurrency:              10,
				Slots:                    100,
				FailureRate:              0.0,
				PayloadSize:              "0kb",
				EventFanout:              1,
				DagSteps:                 1,
				RlKeys:                   0,
				RlLimit:                  0,
				RlDurationUnit:           "",
				AverageDurationThreshold: avgThreshold,
			},
		},
		{
			name: "test for many queued events and little worker throughput",
			config: LoadTestConfig{
				Duration:                 240 * time.Second,
				Events:                   10,
				Delay:                    0 * time.Second,
				WorkerDelay:              120 * time.Second, // will write about 1100 events before the worker is ready
				Wait:                     120 * time.Second,
				Concurrency:              0,
				Slots:                    100,
				FailureRate:              0.0,
				PayloadSize:              "0kb",
				EventFanout:              1,
				DagSteps:                 1,
				RlKeys:                   0,
				RlLimit:                  0,
				RlDurationUnit:           "",
				AverageDurationThreshold: 45 * time.Second, // includes intentional queue wait from WorkerDelay
			},
		},
		{
			name: "test with rate limits",
			config: LoadTestConfig{
				Duration:                 240 * time.Second,
				Events:                   10,
				Delay:                    0 * time.Second,
				Wait:                     60 * time.Second,
				Concurrency:              0,
				Slots:                    100,
				FailureRate:              0.0,
				PayloadSize:              "0kb",
				EventFanout:              1,
				DagSteps:                 1,
				RlKeys:                   10,
				RlLimit:                  100,
				RlDurationUnit:           "second",
				AverageDurationThreshold: avgThreshold,
			},
		},
	}

	if err := harness.WaitEngineReady(t.Context(), startupSleep); err != nil {
		t.Fatalf("failed to bring up engine in time: %s", err)
	}

	for _, tt := range tests {
		tt := tt // pin the loop variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			namespace, err := random.Generate(8)

			if err != nil {
				t.Fatalf("could not generate random namespace: %s", err)
			}

			testConfig := tt.config
			testConfig.Namespace = namespace

			if tt.dagOperator {
				enableDAGOperator(t)
			}

			if err := do(testConfig); (err != nil) != tt.wantErr {
				t.Errorf("do() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.dagOperator {
				// The client joins the namespace and the workflow name with an
				// underscore, and the stored name is lowercased.
				requireDAGOperatorWorkflow(t, namespace+"_"+eventkeys.WorkflowStandardName(0))
			}
		})
	}

	log.Printf("test complete")
}

// enableDAGOperator grants the harness tenant the DAG operator entitlement,
// which registration reads on every call, so workflows registered after this
// point that have more than one task run on the operator.
func enableDAGOperator(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("could not connect to the harness database: %v", err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
		INSERT INTO tenant_entitlement (tenant_id)
		SELECT id FROM "Tenant"
		ON CONFLICT (tenant_id) DO NOTHING`)
	if err != nil {
		t.Fatalf("could not create tenant entitlements: %v", err)
	}

	_, err = conn.Exec(ctx, `UPDATE tenant_entitlement SET dag_operator = TRUE`)
	if err != nil {
		t.Fatalf("could not enable the DAG operator entitlement: %v", err)
	}
}

// requireDAGOperatorWorkflow fails the test when the latest version of the
// named workflow was not registered on the DAG operator, so a run that fell
// back to the classic DAG path cannot pass as an operator run.
func requireDAGOperatorWorkflow(t *testing.T, workflowName string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("could not connect to the harness database: %v", err)
	}
	defer conn.Close(ctx)

	var usingDAGOperator bool
	err = conn.QueryRow(ctx, `
		SELECT wv."isUsingDagOperator"
		FROM "WorkflowVersion" wv
		JOIN "Workflow" w ON w."id" = wv."workflowId"
		WHERE lower(w."name") = lower($1) AND w."deletedAt" IS NULL
		ORDER BY wv."order" DESC
		LIMIT 1`, workflowName).Scan(&usingDAGOperator)
	if err != nil {
		t.Fatalf("could not read the workflow version for %s: %v (%s)", workflowName, err, describeWorkflows(ctx, conn, eventkeys.WorkflowStandardName(0)))
	}

	if !usingDAGOperator {
		t.Fatalf("workflow %s was not registered on the DAG operator", workflowName)
	}
}

// describeWorkflows summarizes the workflows whose name contains fragment, for
// failure messages when the expected workflow cannot be found.
func describeWorkflows(ctx context.Context, conn *pgx.Conn, fragment string) string {
	var db string
	if err := conn.QueryRow(ctx, `SELECT current_database()`).Scan(&db); err != nil {
		return fmt.Sprintf("could not describe workflows: %v", err)
	}

	rows, err := conn.Query(ctx, `
		SELECT w."name", w."tenantId", w."deletedAt" IS NOT NULL, COALESCE(wv."isUsingDagOperator", FALSE)
		FROM "Workflow" w
		LEFT JOIN "WorkflowVersion" wv ON wv."workflowId" = w."id"
		WHERE w."name" ILIKE '%' || $1 || '%'
		ORDER BY w."createdAt" DESC
		LIMIT 10`, fragment)
	if err != nil {
		return fmt.Sprintf("database %s, could not list workflows: %v", db, err)
	}
	defer rows.Close()

	var found []string
	for rows.Next() {
		var name, tenantId string
		var deleted, dagOperator bool
		if err := rows.Scan(&name, &tenantId, &deleted, &dagOperator); err != nil {
			return fmt.Sprintf("database %s, could not scan workflows: %v", db, err)
		}
		found = append(found, fmt.Sprintf("%s tenant=%s deleted=%t dagOperator=%t", name, tenantId, deleted, dagOperator))
	}

	return fmt.Sprintf("database %s, workflows matching %q: %v", db, fragment, found)
}

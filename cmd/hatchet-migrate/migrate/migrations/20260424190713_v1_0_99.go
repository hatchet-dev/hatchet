package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/pressly/goose/v3"
	"golang.org/x/sync/errgroup"
)

func init() {
	goose.AddMigrationNoTxContext(up20260424190713, down20260424190713)
}

const (
	v1TasksOlapTable = "v1_tasks_olap"
	v1DagsOlapTable  = "v1_dags_olap"
)

func buildCreateMirrorTableSQL(oldParent, newParent, colDefs string) string {
	return `
DO $$
DECLARE
	r         	   RECORD;
	partition_date date;
BEGIN
	CREATE TABLE IF NOT EXISTS ` + newParent + ` (
` + colDefs + `,
		PRIMARY KEY (inserted_at, id)
	) PARTITION BY RANGE (inserted_at);

	FOR r IN (
		SELECT c.relname
		FROM pg_class c
		JOIN pg_inherits i ON c.oid  = i.inhrelid
		JOIN pg_class p ON p.oid  = i.inhparent
		WHERE
			p.relname = '` + oldParent + `'
		  	AND c.relkind IN ('r', 'p')
	) LOOP
		partition_date := to_date(
			substring(r.relname from length('` + oldParent + `_') + 1),
			'YYYYMMDD'
		);

		PERFORM create_v1_range_partition('` + newParent + `', partition_date);
	END LOOP;
END;
$$`
}

const v1RunsOlapNewColDefs = `
	tenant_id               UUID NOT NULL,
	id                      BIGINT NOT NULL,
	inserted_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	external_id             UUID NOT NULL DEFAULT gen_random_uuid(),
	readable_status         v1_readable_status_olap NOT NULL DEFAULT 'QUEUED',
	kind                    v1_run_kind NOT NULL,
	workflow_id             UUID NOT NULL,
	workflow_version_id     UUID NOT NULL,
	additional_metadata     JSONB,
	parent_task_external_id UUID
`

const v1RunsOlapMirrorFn = `CREATE OR REPLACE FUNCTION v1_runs_olap_mirror_fn()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
	IF TG_OP = 'INSERT' THEN
		INSERT INTO v1_runs_olap_new (
			tenant_id,
			id,
			inserted_at,
			external_id,
			readable_status,
			kind,
			workflow_id,
			workflow_version_id,
			additional_metadata,
			parent_task_external_id
		) VALUES (
			NEW.tenant_id,
			NEW.id,
			NEW.inserted_at,
			NEW.external_id,
			NEW.readable_status,
			NEW.kind,
			NEW.workflow_id,
			NEW.workflow_version_id,
			NEW.additional_metadata,
			NEW.parent_task_external_id
		)
		ON CONFLICT (inserted_at, id)
		DO UPDATE SET
			tenant_id               = EXCLUDED.tenant_id,
			external_id             = EXCLUDED.external_id,
			readable_status         = EXCLUDED.readable_status,
			kind                    = EXCLUDED.kind,
			workflow_id             = EXCLUDED.workflow_id,
			workflow_version_id     = EXCLUDED.workflow_version_id,
			additional_metadata     = EXCLUDED.additional_metadata,
			parent_task_external_id = EXCLUDED.parent_task_external_id;
		RETURN NEW;
	ELSIF TG_OP = 'UPDATE' THEN
		UPDATE v1_runs_olap_new SET
			tenant_id               = NEW.tenant_id,
			external_id             = NEW.external_id,
			readable_status         = NEW.readable_status,
			kind                    = NEW.kind,
			workflow_id             = NEW.workflow_id,
			workflow_version_id     = NEW.workflow_version_id,
			additional_metadata     = NEW.additional_metadata,
			parent_task_external_id = NEW.parent_task_external_id
		WHERE inserted_at = NEW.inserted_at AND id = NEW.id;
		RETURN NEW;
	ELSIF TG_OP = 'DELETE' THEN
		DELETE FROM v1_runs_olap_new
		WHERE inserted_at = OLD.inserted_at AND id = OLD.id;
		RETURN OLD;
	END IF;
	RETURN NULL;
END;
$$`

const v1RunsOlapMirrorTrigger = `CREATE OR REPLACE TRIGGER v1_runs_olap_mirror
AFTER INSERT OR UPDATE OR DELETE ON v1_runs_olap
FOR EACH ROW EXECUTE FUNCTION v1_runs_olap_mirror_fn()`

const v1TasksOlapNewColDefs = `
		tenant_id               UUID NOT NULL,
		id                      BIGINT NOT NULL,
		inserted_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		external_id             UUID NOT NULL DEFAULT gen_random_uuid(),
		queue                   TEXT NOT NULL,
		action_id               TEXT NOT NULL,
		step_id                 UUID NOT NULL,
		workflow_id             UUID NOT NULL,
		workflow_version_id     UUID NOT NULL,
		workflow_run_id         UUID NOT NULL,
		schedule_timeout        TEXT NOT NULL,
		step_timeout            TEXT,
		priority                INTEGER DEFAULT 1,
		sticky                  v1_sticky_strategy_olap NOT NULL,
		desired_worker_id       UUID,
		display_name            TEXT NOT NULL,
		input                   JSONB NOT NULL,
		additional_metadata     JSONB,
		readable_status         v1_readable_status_olap NOT NULL DEFAULT 'QUEUED',
		latest_retry_count      INT NOT NULL DEFAULT 0,
		latest_worker_id        UUID,
		dag_id                  BIGINT,
		dag_inserted_at         TIMESTAMPTZ,
		parent_task_external_id UUID,
		is_durable 			 	BOOLEAN NOT NULL DEFAULT FALSE`

const v1TasksOlapMirrorFn = `CREATE OR REPLACE FUNCTION v1_tasks_olap_mirror_fn()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
	IF TG_OP = 'INSERT' THEN
		INSERT INTO v1_tasks_olap_new (
			tenant_id,
			id,
			inserted_at,
			external_id,
			queue,
			action_id,
			step_id,
			workflow_id,
			workflow_version_id,
			workflow_run_id,
			schedule_timeout,
			step_timeout,
			priority,
			sticky,
			desired_worker_id,
			display_name,
			input,
			additional_metadata,
			readable_status,
			latest_retry_count,
			latest_worker_id,
			dag_id,
			dag_inserted_at,
			parent_task_external_id,
			is_durable
		) VALUES (
			NEW.tenant_id,
			NEW.id,
			NEW.inserted_at,
			NEW.external_id,
			NEW.queue,
			NEW.action_id,
			NEW.step_id,
			NEW.workflow_id,
			NEW.workflow_version_id,
			NEW.workflow_run_id,
			NEW.schedule_timeout,
			NEW.step_timeout,
			NEW.priority,
			NEW.sticky,
			NEW.desired_worker_id,
			NEW.display_name,
			NEW.input,
			NEW.additional_metadata,
			NEW.readable_status,
			NEW.latest_retry_count,
			NEW.latest_worker_id,
			NEW.dag_id,
			NEW.dag_inserted_at,
			NEW.parent_task_external_id,
			NEW.is_durable
		)
		ON CONFLICT (inserted_at, id)
		DO UPDATE SET
			tenant_id               = EXCLUDED.tenant_id,
			external_id             = EXCLUDED.external_id,
			queue                   = EXCLUDED.queue,
			action_id               = EXCLUDED.action_id,
			step_id                 = EXCLUDED.step_id,
			workflow_id             = EXCLUDED.workflow_id,
			workflow_version_id     = EXCLUDED.workflow_version_id,
			workflow_run_id         = EXCLUDED.workflow_run_id,
			schedule_timeout        = EXCLUDED.schedule_timeout,
			step_timeout            = EXCLUDED.step_timeout,
			priority                = EXCLUDED.priority,
			sticky                  = EXCLUDED.sticky,
			desired_worker_id       = EXCLUDED.desired_worker_id,
			display_name            = EXCLUDED.display_name,
			input                   = EXCLUDED.input,
			additional_metadata     = EXCLUDED.additional_metadata,
			readable_status         = EXCLUDED.readable_status,
			latest_retry_count      = EXCLUDED.latest_retry_count,
			latest_worker_id        = EXCLUDED.latest_worker_id,
			dag_id                  = EXCLUDED.dag_id,
			dag_inserted_at         = EXCLUDED.dag_inserted_at,
			parent_task_external_id = EXCLUDED.parent_task_external_id,
			is_durable				= EXCLUDED.is_durable;
		RETURN NEW;
	ELSIF TG_OP = 'UPDATE' THEN
		UPDATE v1_tasks_olap_new SET
			tenant_id               = NEW.tenant_id,
			external_id             = NEW.external_id,
			queue                   = NEW.queue,
			action_id               = NEW.action_id,
			step_id                 = NEW.step_id,
			workflow_id             = NEW.workflow_id,
			workflow_version_id     = NEW.workflow_version_id,
			workflow_run_id         = NEW.workflow_run_id,
			schedule_timeout        = NEW.schedule_timeout,
			step_timeout            = NEW.step_timeout,
			priority                = NEW.priority,
			sticky                  = NEW.sticky,
			desired_worker_id       = NEW.desired_worker_id,
			display_name            = NEW.display_name,
			input                   = NEW.input,
			additional_metadata     = NEW.additional_metadata,
			readable_status         = NEW.readable_status,
			latest_retry_count      = NEW.latest_retry_count,
			latest_worker_id        = NEW.latest_worker_id,
			dag_id                  = NEW.dag_id,
			dag_inserted_at         = NEW.dag_inserted_at,
			parent_task_external_id = NEW.parent_task_external_id,
			is_durable				= NEW.is_durable
		WHERE inserted_at = NEW.inserted_at AND id = NEW.id;
		RETURN NEW;
	ELSIF TG_OP = 'DELETE' THEN
		DELETE FROM v1_tasks_olap_new
		WHERE inserted_at = OLD.inserted_at AND id = OLD.id;
		RETURN OLD;
	END IF;
	RETURN NULL;
END;
$$`

const v1TasksOlapMirrorTrigger = `CREATE OR REPLACE TRIGGER v1_tasks_olap_mirror
AFTER INSERT OR UPDATE OR DELETE ON v1_tasks_olap
FOR EACH ROW EXECUTE FUNCTION v1_tasks_olap_mirror_fn()`

const v1DagsOlapNewColDefs = `
		id                      BIGINT NOT NULL,
		inserted_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		tenant_id               UUID NOT NULL,
		external_id             UUID NOT NULL,
		display_name            TEXT NOT NULL,
		workflow_id             UUID NOT NULL,
		workflow_version_id     UUID NOT NULL,
		readable_status         v1_readable_status_olap NOT NULL DEFAULT 'QUEUED',
		input                   JSONB NOT NULL,
		additional_metadata     JSONB,
		parent_task_external_id UUID,
		total_tasks             INT NOT NULL DEFAULT 1`

const v1DagsOlapMirrorFn = `CREATE OR REPLACE FUNCTION v1_dags_olap_mirror_fn()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
	IF TG_OP = 'INSERT' THEN
		INSERT INTO v1_dags_olap_new (
			id,
			inserted_at,
			tenant_id,
			external_id,
			display_name,
			workflow_id,
			workflow_version_id,
			readable_status,
			input,
			additional_metadata,
			parent_task_external_id,
			total_tasks
		) VALUES (
			NEW.id,
			NEW.inserted_at,
			NEW.tenant_id,
			NEW.external_id,
			NEW.display_name,
			NEW.workflow_id,
			NEW.workflow_version_id,
			NEW.readable_status,
			NEW.input,
			NEW.additional_metadata,
			NEW.parent_task_external_id,
			NEW.total_tasks
		)
		ON CONFLICT (inserted_at, id)
		DO UPDATE SET
			tenant_id               = EXCLUDED.tenant_id,
			external_id             = EXCLUDED.external_id,
			display_name            = EXCLUDED.display_name,
			workflow_id             = EXCLUDED.workflow_id,
			workflow_version_id     = EXCLUDED.workflow_version_id,
			readable_status         = EXCLUDED.readable_status,
			input                   = EXCLUDED.input,
			additional_metadata     = EXCLUDED.additional_metadata,
			parent_task_external_id = EXCLUDED.parent_task_external_id,
			total_tasks             = EXCLUDED.total_tasks;
		RETURN NEW;
	ELSIF TG_OP = 'UPDATE' THEN
		UPDATE v1_dags_olap_new SET
			tenant_id               = NEW.tenant_id,
			external_id             = NEW.external_id,
			display_name            = NEW.display_name,
			workflow_id             = NEW.workflow_id,
			workflow_version_id     = NEW.workflow_version_id,
			readable_status         = NEW.readable_status,
			input                   = NEW.input,
			additional_metadata     = NEW.additional_metadata,
			parent_task_external_id = NEW.parent_task_external_id,
			total_tasks             = NEW.total_tasks
		WHERE inserted_at = NEW.inserted_at AND id = NEW.id;
		RETURN NEW;
	ELSIF TG_OP = 'DELETE' THEN
		DELETE FROM v1_dags_olap_new
		WHERE inserted_at = OLD.inserted_at AND id = OLD.id;
		RETURN OLD;
	END IF;
	RETURN NULL;
END;
$$`

const v1DagsOlapMirrorTrigger = `CREATE OR REPLACE TRIGGER v1_dags_olap_mirror
AFTER INSERT OR UPDATE OR DELETE ON v1_dags_olap
FOR EACH ROW EXECUTE FUNCTION v1_dags_olap_mirror_fn()`

var v1RunsOlapCols = []string{
	"tenant_id",
	"id",
	"inserted_at",
	"external_id",
	"readable_status",
	"kind",
	"workflow_id",
	"workflow_version_id",
	"additional_metadata",
	"parent_task_external_id",
}

var v1TasksOlapCols = []string{
	"tenant_id",
	"id",
	"inserted_at",
	"external_id",
	"queue",
	"action_id",
	"step_id",
	"workflow_id",
	"workflow_version_id",
	"workflow_run_id",
	"schedule_timeout",
	"step_timeout",
	"priority",
	"sticky",
	"desired_worker_id",
	"display_name",
	"input",
	"additional_metadata",
	"readable_status",
	"latest_retry_count",
	"latest_worker_id",
	"dag_id",
	"dag_inserted_at",
	"parent_task_external_id",
	"is_durable",
}

var v1DagsOlapCols = []string{
	"id",
	"inserted_at",
	"tenant_id",
	"external_id",
	"display_name",
	"workflow_id",
	"workflow_version_id",
	"readable_status",
	"input",
	"additional_metadata",
	"parent_task_external_id",
	"total_tasks",
}

func backfillByPartition(ctx context.Context, db *sql.DB, srcTable, newTable string, cols []string) error {
	partitions, err := listLeafPartitions(ctx, db, srcTable, 1)

	if err != nil {
		return fmt.Errorf("list partitions for %s: %w", srcTable, err)
	}

	colList := strings.Join(cols, ", ")

	srcCols := make([]string, len(cols))
	for i, c := range cols {
		srcCols[i] = "src." + c
	}

	srcColList := strings.Join(srcCols, ", ")

	for _, partition := range partitions {
		var alreadyBackfilled bool
		if err := db.QueryRowContext(
			ctx,
			`
			SELECT EXISTS(
				SELECT 1 FROM v1_olap_backfill_progress
				WHERE table_name = $1 AND partition_name = $2
			)
			`,
			srcTable,
			partition,
		).Scan(&alreadyBackfilled); err != nil {
			return fmt.Errorf("check progress for partition %s: %w", partition, err)
		}

		if alreadyBackfilled {
			continue
		}

		insertSQL := fmt.Sprintf(
			"INSERT INTO %s (%s) SELECT %s FROM %s src ON CONFLICT DO NOTHING",
			newTable, colList, srcColList, partition,
		)

		if _, err := db.ExecContext(ctx, insertSQL); err != nil {
			return fmt.Errorf("backfill partition %s into %s: %w", partition, newTable, err)
		}

		if _, err := db.ExecContext(ctx, `
			INSERT INTO v1_olap_backfill_progress (table_name, partition_name)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, srcTable, partition); err != nil {
			return fmt.Errorf("record progress for partition %s: %w", partition, err)
		}
	}

	return nil
}

func up20260424190713(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS v1_olap_backfill_progress (
			table_name     TEXT NOT NULL,
			partition_name TEXT NOT NULL,
			completed_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (table_name, partition_name)
		)
	`); err != nil {
		return fmt.Errorf("create backfill progress table: %w", err)
	}

	eg := &errgroup.Group{}

	eg.Go(func() error {
		if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS ix_v1_runs_olap_tenant_id`); err != nil {
			return fmt.Errorf("drop old index on %s: %w", v1RunsOlapTable, err)
		}

		if _, err := db.ExecContext(ctx, buildCreateMirrorTableSQL(v1RunsOlapTable, v1RunsOlapTable+"_new", v1RunsOlapNewColDefs)); err != nil {
			return fmt.Errorf("create %s_new: %w", v1RunsOlapTable, err)
		}

		if _, err := db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS ix_v1_runs_olap_new_parent_task_external_id ON v1_runs_olap_new (parent_task_external_id) WHERE parent_task_external_id IS NOT NULL"); err != nil {
			return fmt.Errorf("failed to create index ix_v1_runs_olap_new_parent_task_external_id: %w", err)
		}

		if _, err := db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS ix_v1_runs_olap_new_tenant_status_ins_at ON v1_runs_olap_new (tenant_id, readable_status, inserted_at DESC);"); err != nil {
			return fmt.Errorf("failed to create index ix_v1_runs_olap_new_tenant_status_ins_at: %w", err)
		}

		if _, err := db.ExecContext(ctx, v1RunsOlapMirrorFn); err != nil {
			return fmt.Errorf("create mirror function for %s: %w", v1RunsOlapTable, err)
		}
		if _, err := db.ExecContext(ctx, v1RunsOlapMirrorTrigger); err != nil {
			return fmt.Errorf("create mirror trigger for %s: %w", v1RunsOlapTable, err)
		}

		if err := backfillByPartition(ctx, db, v1RunsOlapTable, v1RunsOlapTable+"_new", v1RunsOlapCols); err != nil {
			return fmt.Errorf("backfill %s_new: %w", v1RunsOlapTable, err)
		}

		return nil
	})

	eg.Go(func() error {
		if _, err := db.ExecContext(ctx, buildCreateMirrorTableSQL(v1TasksOlapTable, v1TasksOlapTable+"_new", v1TasksOlapNewColDefs)); err != nil {
			return fmt.Errorf("create %s_new: %w", v1TasksOlapTable, err)
		}

		if _, err := db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS v1_tasks_olap_new_workflow_id_idx ON v1_tasks_olap_new (tenant_id, workflow_id)"); err != nil {
			return fmt.Errorf("failed to create index v1_tasks_olap_new_workflow_id_idx: %w", err)
		}
		if _, err := db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS v1_tasks_olap_new_worker_id_idx ON v1_tasks_olap_new (tenant_id, latest_worker_id) WHERE latest_worker_id IS NOT NULL"); err != nil {
			return fmt.Errorf("failed to create index v1_tasks_olap_new_worker_id_idx: %w", err)
		}

		if _, err := db.ExecContext(ctx, v1TasksOlapMirrorFn); err != nil {
			return fmt.Errorf("create mirror function for %s: %w", v1TasksOlapTable, err)
		}
		if _, err := db.ExecContext(ctx, v1TasksOlapMirrorTrigger); err != nil {
			return fmt.Errorf("create mirror trigger for %s: %w", v1TasksOlapTable, err)
		}

		if err := backfillByPartition(ctx, db, v1TasksOlapTable, v1TasksOlapTable+"_new", v1TasksOlapCols); err != nil {
			return fmt.Errorf("backfill %s_new: %w", v1TasksOlapTable, err)
		}

		return nil
	})

	eg.Go(func() error {
		if _, err := db.ExecContext(ctx, buildCreateMirrorTableSQL(v1DagsOlapTable, v1DagsOlapTable+"_new", v1DagsOlapNewColDefs)); err != nil {
			return fmt.Errorf("create %s_new: %w", v1DagsOlapTable, err)
		}

		if _, err := db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS v1_dags_olap_new_workflow_id_idx ON v1_dags_olap_new (tenant_id, workflow_id)"); err != nil {
			return fmt.Errorf("failed to create index v1_dags_olap_new_workflow_id_idx: %w", err)
		}

		if _, err := db.ExecContext(ctx, v1DagsOlapMirrorFn); err != nil {
			return fmt.Errorf("create mirror function for %s: %w", v1DagsOlapTable, err)
		}
		if _, err := db.ExecContext(ctx, v1DagsOlapMirrorTrigger); err != nil {
			return fmt.Errorf("create mirror trigger for %s: %w", v1DagsOlapTable, err)
		}

		if err := backfillByPartition(ctx, db, v1DagsOlapTable, v1DagsOlapTable+"_new", v1DagsOlapCols); err != nil {
			return fmt.Errorf("backfill %s_new: %w", v1DagsOlapTable, err)
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	for _, table := range []struct{ src, dst string }{
		{v1RunsOlapTable, v1RunsOlapTable + "_new"},
		{v1TasksOlapTable, v1TasksOlapTable + "_new"},
		{v1DagsOlapTable, v1DagsOlapTable + "_new"},
	} {
		var newCount, existingCount int64

		if err := db.QueryRowContext(ctx, fmt.Sprintf(`
			WITH counts AS (
				SELECT
					(SELECT COUNT(*) FROM %s) AS new_count,
					(SELECT COUNT(*) FROM %s) AS existing_count
			)
			SELECT new_count, existing_count
			FROM counts
		`, table.dst, table.src)).Scan(&newCount, &existingCount); err != nil {
			return fmt.Errorf("counting rows in %s and %s: %w", table.dst, table.src, err)
		}

		if math.Abs(float64(newCount)-float64(existingCount)) > 1000 {
			return fmt.Errorf("row count mismatch after backfill for %s: new=%d, existing=%d", table.src, newCount, existingCount)
		}
	}

	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS v1_olap_backfill_progress`); err != nil {
		return fmt.Errorf("drop backfill progress table: %w", err)
	}

	return nil
}

func down20260424190713(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS v1_olap_backfill_progress`); err != nil {
		return fmt.Errorf("drop backfill progress table: %w", err)
	}

	for _, table := range []string{v1RunsOlapTable, v1TasksOlapTable, v1DagsOlapTable} {
		if _, err := db.ExecContext(ctx, fmt.Sprintf(
			`DROP TRIGGER IF EXISTS %s_mirror ON %s`,
			table, table,
		)); err != nil {
			return fmt.Errorf("drop trigger on %s: %w", table, err)
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf(
			`DROP FUNCTION IF EXISTS %s_mirror_fn()`,
			table,
		)); err != nil {
			return fmt.Errorf("drop mirror function for %s: %w", table, err)
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf(
			`DROP TABLE IF EXISTS %s_new CASCADE`,
			table,
		)); err != nil {
			return fmt.Errorf("drop table %s_new: %w", table, err)
		}
	}

	// intentionally not recreating / re-dropping indexes in the down, since we use `IF NOT EXISTS`
	// in the up migration anyways

	return nil
}

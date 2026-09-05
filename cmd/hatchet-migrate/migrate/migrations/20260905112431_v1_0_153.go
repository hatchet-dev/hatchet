package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upV10153, downV10153)
}

const v1PayloadNewCreateTableSQL = `
CREATE TABLE IF NOT EXISTS v1_payload_new (
	tenant_id UUID NOT NULL,
	id BIGINT NOT NULL,
	inserted_at TIMESTAMPTZ NOT NULL,
	inserted_at_date DATE NOT NULL DEFAULT CURRENT_TIMESTAMP::DATE,
	external_id UUID NOT NULL DEFAULT gen_random_uuid(),
	type v1_payload_type NOT NULL,
	location v1_payload_location NOT NULL,
	external_location_key TEXT,
	inline_content JSONB,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

	PRIMARY KEY (external_id, inserted_at_date),
	-- explicitly named because ATTACH PARTITION matches check constraints by name, so the
	-- offload swap breaks if the auto-generated name drifts between parent and temp tables
	CONSTRAINT v1_payload_location_check CHECK (
		location = 'INLINE'
		OR
		(location = 'EXTERNAL' AND inline_content IS NULL AND external_location_key IS NOT NULL)
	)
) PARTITION BY RANGE(inserted_at_date)`

const v1PayloadMirrorFnSQL = `CREATE OR REPLACE FUNCTION v1_payload_mirror_fn()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
	IF TG_OP = 'INSERT' THEN
		INSERT INTO v1_payload_new (
			tenant_id,
			id,
			inserted_at,
			inserted_at_date,
			external_id,
			type,
			location,
			external_location_key,
			inline_content,
			updated_at
		) VALUES (
			NEW.tenant_id,
			NEW.id,
			NEW.inserted_at,
			NEW.inserted_at_date,
			NEW.external_id,
			NEW.type,
			NEW.location,
			NEW.external_location_key,
			NEW.inline_content,
			NEW.updated_at
		)
		ON CONFLICT (external_id, inserted_at_date)
		DO UPDATE SET
			tenant_id             = EXCLUDED.tenant_id,
			id                    = EXCLUDED.id,
			inserted_at           = EXCLUDED.inserted_at,
			type                  = EXCLUDED.type,
			location              = EXCLUDED.location,
			external_location_key = EXCLUDED.external_location_key,
			inline_content        = EXCLUDED.inline_content,
			updated_at            = EXCLUDED.updated_at;
		RETURN NEW;
	ELSIF TG_OP = 'UPDATE' THEN
		UPDATE v1_payload_new SET
			tenant_id             = NEW.tenant_id,
			id                    = NEW.id,
			inserted_at           = NEW.inserted_at,
			type                  = NEW.type,
			location              = NEW.location,
			external_location_key = NEW.external_location_key,
			inline_content        = NEW.inline_content,
			updated_at            = NEW.updated_at
		WHERE external_id = NEW.external_id AND inserted_at_date = NEW.inserted_at_date;
		RETURN NEW;
	ELSIF TG_OP = 'DELETE' THEN
		DELETE FROM v1_payload_new
		WHERE external_id = OLD.external_id AND inserted_at_date = OLD.inserted_at_date;
		RETURN OLD;
	END IF;
	RETURN NULL;
END;
$$`

const v1PayloadMirrorTriggerSQL = `CREATE OR REPLACE TRIGGER v1_payload_mirror
AFTER INSERT OR UPDATE OR DELETE ON v1_payload
FOR EACH ROW EXECUTE FUNCTION v1_payload_mirror_fn()`

var v1PayloadCols = []string{
	"tenant_id",
	"id",
	"inserted_at",
	"inserted_at_date",
	"external_id",
	"type",
	"location",
	"external_location_key",
	"inline_content",
	"updated_at",
}

func v1PayloadPartitionDates(ctx context.Context, db *sql.DB) ([]time.Time, error) {
	partitions, err := listLeafPartitions(ctx, db, "v1_payload", 1)

	if err != nil {
		return nil, err
	}

	dates := make([]time.Time, 0, len(partitions))

	for _, partition := range partitions {
		suffix := partition[strings.LastIndex(partition, "_")+1:]
		date, err := time.Parse("20060102", suffix)

		if err != nil {
			return nil, fmt.Errorf("could not parse partition date from %s: %w", partition, err)
		}

		dates = append(dates, date)
	}

	return dates, nil
}

func upV10153(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, v1PayloadNewCreateTableSQL); err != nil {
		return fmt.Errorf("create v1_payload_new: %w", err)
	}

	partitionDates, err := v1PayloadPartitionDates(ctx, db)

	if err != nil {
		return err
	}

	for _, date := range partitionDates {
		if _, err := db.ExecContext(ctx, "SELECT create_v1_range_partition('v1_payload_new', $1::date)", date.Format("2006-01-02")); err != nil {
			return fmt.Errorf("create v1_payload_new partition for %s: %w", date.Format("2006-01-02"), err)
		}
	}

	if _, err := db.ExecContext(ctx, v1PayloadMirrorFnSQL); err != nil {
		return fmt.Errorf("create mirror function: %w", err)
	}

	if _, err := db.ExecContext(ctx, v1PayloadMirrorTriggerSQL); err != nil {
		return fmt.Errorf("create mirror trigger: %w", err)
	}

	partitions, err := listLeafPartitions(ctx, db, "v1_payload", 1)

	if err != nil {
		return fmt.Errorf("list v1_payload partitions: %w", err)
	}

	colList := strings.Join(v1PayloadCols, ", ")

	for _, partition := range partitions {
		// #nosec G201 -- table/column identifiers are derived from internal migration logic, not user input
		insertSQL := fmt.Sprintf(
			"INSERT INTO v1_payload_new (%s) SELECT %s FROM %s ON CONFLICT DO NOTHING",
			colList, colList, quoteIdent(partition),
		)

		if _, err := db.ExecContext(ctx, insertSQL); err != nil {
			return fmt.Errorf("backfill partition %s: %w", partition, err)
		}
	}

	var newCount, existingCount int64

	if err := db.QueryRowContext(ctx, `
		WITH counts AS (
			SELECT
				(SELECT COUNT(*) FROM v1_payload_new) AS new_count,
				(SELECT COUNT(*) FROM v1_payload) AS existing_count
		)
		SELECT new_count, existing_count
		FROM counts
	`).Scan(&newCount, &existingCount); err != nil {
		return fmt.Errorf("counting rows in v1_payload_new and v1_payload: %w", err)
	}

	if newCount != existingCount {
		return fmt.Errorf("row count mismatch after backfill: new=%d, existing=%d", newCount, existingCount)
	}

	return nil
}

func downV10153(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `DROP TRIGGER IF EXISTS v1_payload_mirror ON v1_payload`); err != nil {
		return fmt.Errorf("drop mirror trigger: %w", err)
	}

	if _, err := db.ExecContext(ctx, `DROP FUNCTION IF EXISTS v1_payload_mirror_fn()`); err != nil {
		return fmt.Errorf("drop mirror function: %w", err)
	}

	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS v1_payload_new CASCADE`); err != nil {
		return fmt.Errorf("drop v1_payload_new: %w", err)
	}

	return nil
}

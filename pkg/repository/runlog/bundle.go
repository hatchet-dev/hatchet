// Package runlog is the object-store log of finished work that core Postgres has dropped. Partition
// maintenance writes one bundle per task when a day partition is archived (see
// TaskRepositoryImpl.UpdateTablePartitionsAt); readers fall back to it when core has no row.
//
// Layout, per tenant per UTC day, under <prefix>/v1/<tenant_id>/<YYYY>/<MM>/<DD>/:
//
//	data-<NNNNN>.jsonl.gz  bundles, each one JSON object compressed as its own gzip member, so one bundle
//	                       is fetched with a ranged GET and decompressed alone
//	index.json.gz          written after every data file: task, run and payload external id -> location
//	_COMPLETE              zero-byte marker written after the index; a tenant-day without it is absent
package runlog

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// FormatVersion is written into every bundle and index. Bump it when a field changes meaning.
const FormatVersion = 1

// Bundle is one task with what the readers of finished work need. Rows are the core rows as sqlc reads
// them, so a reader can treat them like rows read from core.
type Bundle struct {
	FormatVersion int       `json:"format_version"`
	TenantID      uuid.UUID `json:"tenant_id"`

	Task   *sqlcv1.V1Task        `json:"task"`
	Events []*sqlcv1.V1TaskEvent `json:"events"`

	// Payloads are the resolved bodies of the task input, every event's data, the durable event log entries'
	// data and results, and the DAG input when Dag is set. A payload shared by many tasks (one event input,
	// one DAG input) is copied into each bundle that references it.
	Payloads []Payload `json:"payloads"`

	// DurableLog is set for durable tasks and DAG orchestrators.
	DurableLog *DurableLog `json:"durable_log,omitempty"`

	// Dag is set on one bundle per legacy DAG: the DAG's lowest task id among its tasks inserted on the
	// DAG's own day.
	Dag *Dag `json:"dag,omitempty"`
}

// Payload is one payload body as the payload store returns it for a referrer.
type Payload struct {
	Type sqlcv1.V1PayloadType `json:"type"`

	// ID and InsertedAt are the referrer's key as passed to the payload store: the task, event, DAG or
	// durable log entry the payload belongs to.
	ID         int64              `json:"id"`
	InsertedAt pgtype.Timestamptz `json:"inserted_at"`

	// ExternalID is the external_id of the v1_payload row read: the payload reference id when the row is
	// shared, otherwise the referrer's own external id.
	ExternalID uuid.UUID `json:"external_id"`

	// Shared is set when the row was read through a payload reference id.
	Shared bool `json:"shared,omitempty"`

	Value json.RawMessage `json:"value"`
}

type DurableLog struct {
	File         *sqlcv1.V1DurableEventLogFile          `json:"file,omitempty"`
	Entries      []*sqlcv1.V1DurableEventLogEntry       `json:"entries"`
	BranchPoints []*sqlcv1.V1DurableEventLogBranchPoint `json:"branch_points"`
}

type Dag struct {
	Dag   *sqlcv1.V1Dag         `json:"dag"`
	Data  *sqlcv1.V1DagData     `json:"data,omitempty"`
	Tasks []*sqlcv1.V1DagToTask `json:"tasks"`
}

// PayloadByExternalID returns the bundle's payload stored under the given v1_payload external id.
func (b *Bundle) PayloadByExternalID(externalID uuid.UUID) *Payload {
	for i := range b.Payloads {
		if b.Payloads[i].ExternalID == externalID {
			return &b.Payloads[i]
		}
	}

	return nil
}

// PayloadFor returns the bundle's payload of the given type for a referrer (id, inserted_at).
func (b *Bundle) PayloadFor(typ sqlcv1.V1PayloadType, id int64, insertedAt pgtype.Timestamptz) *Payload {
	for i := range b.Payloads {
		p := &b.Payloads[i]

		if p.Type == typ && p.ID == id && p.InsertedAt.Time.Equal(insertedAt.Time) {
			return p
		}
	}

	return nil
}

package runlog

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

// maxDataFileBytes is the compressed size at which a data file is uploaded and the next one started.
var maxDataFileBytes = 8 << 20

// Location is where one bundle sits: a gzip member in a data file.
type Location struct {
	File   int   `json:"f"`
	Offset int64 `json:"o"`
	Length int64 `json:"l"`
}

type RunEntry struct {
	// Tasks are the run's tasks archived on this tenant-day.
	Tasks []uuid.UUID `json:"tasks"`
	// OtherDays lists the other days (YYYY-MM-DD) that hold tasks of the run, as far as core knew when this
	// day was archived: the legacy DAG's v1_dag_to_task rows, or the children a DAG orchestrator spawned.
	OtherDays []string `json:"other_days,omitempty"`
}

// Index maps external ids to bundle locations for one tenant-day.
type Index struct {
	FormatVersion int       `json:"format_version"`
	TenantID      uuid.UUID `json:"tenant_id"`
	Day           string    `json:"day"`
	Files         int       `json:"files"`

	Tasks map[uuid.UUID]Location  `json:"tasks"`
	Runs  map[uuid.UUID]*RunEntry `json:"runs"`
	// Payloads maps a v1_payload external id (the payload reference id for shared rows) to the first
	// bundle of the day that holds the payload.
	Payloads map[uuid.UUID]Location `json:"payloads"`
}

// WriteStats are the sizes of what a DayWriter uploaded.
type WriteStats struct {
	Bundles           int64
	Files             int64
	BytesCompressed   int64
	BytesUncompressed int64
}

func (s *WriteStats) Add(o WriteStats) {
	s.Bundles += o.Bundles
	s.Files += o.Files
	s.BytesCompressed += o.BytesCompressed
	s.BytesUncompressed += o.BytesUncompressed
}

// DayWriter writes the archive of one tenant-day. Re-running it for the same tenant-day overwrites the same
// keys; only Close makes the result visible to readers.
type DayWriter struct {
	log    *Log
	tenant uuid.UUID
	day    time.Time

	file  bytes.Buffer
	gz    *gzip.Writer
	index Index
	stats WriteStats
}

func (l *Log) NewDayWriter(tenant uuid.UUID, day time.Time) *DayWriter {
	day = Day(day)

	return &DayWriter{
		log:    l,
		tenant: tenant,
		day:    day,
		gz:     gzip.NewWriter(nil),
		index: Index{
			FormatVersion: FormatVersion,
			TenantID:      tenant,
			Day:           day.Format(time.DateOnly),
			Tasks:         make(map[uuid.UUID]Location),
			Runs:          make(map[uuid.UUID]*RunEntry),
			Payloads:      make(map[uuid.UUID]Location),
		},
	}
}

// Add appends a bundle to the current data file, uploading the file once it is large enough.
func (w *DayWriter) Add(ctx context.Context, b *Bundle) error {
	if b.TenantID != w.tenant {
		panic(fmt.Sprintf("bundle of tenant %s added to the writer of tenant %s", b.TenantID, w.tenant))
	}

	b.FormatVersion = FormatVersion

	raw, err := json.Marshal(b)

	if err != nil {
		return fmt.Errorf("could not encode bundle of task %s: %w", b.Task.ExternalID, err)
	}

	offset := int64(w.file.Len())

	w.gz.Reset(&w.file)

	if _, err := w.gz.Write(raw); err != nil {
		return err
	}

	if err := w.gz.Close(); err != nil {
		return err
	}

	loc := Location{File: w.index.Files, Offset: offset, Length: int64(w.file.Len()) - offset}

	w.index.Tasks[b.Task.ExternalID] = loc

	run := w.runEntry(b.Task.WorkflowRunID)
	run.Tasks = append(run.Tasks, b.Task.ExternalID)

	for _, p := range b.Payloads {
		if _, ok := w.index.Payloads[p.ExternalID]; !ok {
			w.index.Payloads[p.ExternalID] = loc
		}
	}

	w.stats.Bundles++
	w.stats.BytesUncompressed += int64(len(raw))

	if w.file.Len() >= maxDataFileBytes {
		return w.flush(ctx)
	}

	return nil
}

// AddRunDays records other days holding tasks of a run.
func (w *DayWriter) AddRunDays(runExternalID uuid.UUID, days ...time.Time) {
	run := w.runEntry(runExternalID)

	for _, d := range days {
		s := Day(d).Format(time.DateOnly)

		if s != w.index.Day && !slices.Contains(run.OtherDays, s) {
			run.OtherDays = append(run.OtherDays, s)
		}
	}

	slices.Sort(run.OtherDays)
}

func (w *DayWriter) runEntry(runExternalID uuid.UUID) *RunEntry {
	run, ok := w.index.Runs[runExternalID]

	if !ok {
		run = &RunEntry{}
		w.index.Runs[runExternalID] = run
	}

	return run
}

func (w *DayWriter) flush(ctx context.Context) error {
	if w.file.Len() == 0 {
		return nil
	}

	if err := w.log.store.Put(ctx, w.log.dataKey(w.tenant, w.day, w.index.Files), w.file.Bytes()); err != nil {
		return err
	}

	w.stats.Files++
	w.stats.BytesCompressed += int64(w.file.Len())
	w.index.Files++
	w.file.Reset()

	return nil
}

// Close uploads the last data file, then the index, then the _COMPLETE marker.
func (w *DayWriter) Close(ctx context.Context) (WriteStats, error) {
	if err := w.flush(ctx); err != nil {
		return w.stats, err
	}

	raw, err := json.Marshal(&w.index)

	if err != nil {
		return w.stats, err
	}

	var buf bytes.Buffer

	w.gz.Reset(&buf)

	if _, err := w.gz.Write(raw); err != nil {
		return w.stats, err
	}

	if err := w.gz.Close(); err != nil {
		return w.stats, err
	}

	if err := w.log.store.Put(ctx, w.log.indexKey(w.tenant, w.day), buf.Bytes()); err != nil {
		return w.stats, err
	}

	w.stats.BytesCompressed += int64(buf.Len())
	w.stats.BytesUncompressed += int64(len(raw))

	if err := w.log.store.Put(ctx, w.log.completeKey(w.tenant, w.day), nil); err != nil {
		return w.stats, err
	}

	return w.stats, nil
}

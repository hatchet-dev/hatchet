package main

// Track is one load shape and engine pool configuration. Every field is a
// plain value so two tracks compare with ==: compare refuses results recorded
// with different settings than its own.
type Track struct {
	Name string `json:"name"`

	// Engine database pool: DATABASE_MAX_CONNS and the fairpool caps.
	MaxConns         int `json:"max_conns"`
	TenantMaxPercent int `json:"tenant_max_percent"`
	SharedMaxPercent int `json:"shared_max_percent"`

	// Large tenants come first in tenant order, then Small ones.
	Large TenantShape `json:"large"`
	Small TenantShape `json:"small"`

	// TaskMs is how long a backlog task runs; probe tasks return at once.
	TaskMs int `json:"task_ms"`

	// Load starts, runs WarmupS unmeasured, then SIGTERMs the engine every
	// PeriodS; it stops PeriodS after the last SIGTERM.
	WarmupS float64 `json:"warmup_s"`
	PeriodS float64 `json:"period_s"`

	// A probe's API call is bad past APIDeadlineS (it is also the RPC's
	// deadline), its trigger→assigned past AssignDeadlineS.
	APIDeadlineS    float64 `json:"api_deadline_s"`
	AssignDeadlineS float64 `json:"assign_deadline_s"`

	// WindowS is the length of the window after each SIGTERM that the
	// disruption metric covers.
	WindowS float64 `json:"window_s"`

	// NoiseFloorS is how far two runs of the same build can differ in mean
	// lost seconds; a delta has to exceed it to count.
	NoiseFloorS float64 `json:"noise_floor_s"`
}

// TenantShape is the load of Count identical tenants. Each tenant has
// Strategies standalone tasks, each with one task-level GROUP_ROUND_ROBIN
// strategy (max_runs 1) keyed on an input field. The first Backlogged
// strategies keep Backlog tasks queued on one shared key; probes use a fresh
// key, ProbesPerS per tenant, cycling through all the strategies.
type TenantShape struct {
	Count      int     `json:"count"`
	Strategies int     `json:"strategies"`
	Backlogged int     `json:"backlogged"`
	Backlog    int     `json:"backlog"`
	ProbesPerS float64 `json:"probes_per_s"`
}

// tracks are the default tracks, in report order.
var tracks = []Track{
	{
		Name: "single-tenant-default", MaxConns: 50, TenantMaxPercent: 100, SharedMaxPercent: 100,
		Large:  TenantShape{Count: 1, Strategies: 50, Backlogged: 8, Backlog: 20, ProbesPerS: 10},
		TaskMs: 200, WarmupS: 20, PeriodS: 45, APIDeadlineS: 1, AssignDeadlineS: 2,
		WindowS: 30, NoiseFloorS: 1.5,
	},
	{
		Name: "mt-many-equal", MaxConns: 40, TenantMaxPercent: 25, SharedMaxPercent: 50,
		Small:  TenantShape{Count: 25, Strategies: 3, Backlogged: 3, Backlog: 20, ProbesPerS: 2},
		TaskMs: 200, WarmupS: 20, PeriodS: 45, APIDeadlineS: 1, AssignDeadlineS: 2,
		WindowS: 30, NoiseFloorS: 0.5,
	},
	{
		Name: "mt-skewed", MaxConns: 40, TenantMaxPercent: 25, SharedMaxPercent: 50,
		Large:  TenantShape{Count: 1, Strategies: 2000, Backlogged: 5, Backlog: 50, ProbesPerS: 10},
		Small:  TenantShape{Count: 24, Strategies: 2, ProbesPerS: 2},
		TaskMs: 200, WarmupS: 20, PeriodS: 45, APIDeadlineS: 1, AssignDeadlineS: 2,
		WindowS: 30, NoiseFloorS: 1.5,
	},
}

func trackByName(name string) (Track, bool) {
	for _, t := range tracks {
		if t.Name == name {
			return t, true
		}
	}
	return Track{}, false
}

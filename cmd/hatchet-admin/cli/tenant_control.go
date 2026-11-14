package cli

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

var (
	tenantControlTenantID      string
	tenantControlPaused        bool
	tenantControlDBConnPercent int32
	tenantControlDBTimePercent int32
	tenantControlIngest        int32
	tenantControlReason        string
)

var tenantControlCmd = &cobra.Command{
	Use:   "tenant-control",
	Short: "command for operator controls on a tenant: pause, an ingest rate limit, a database time budget and slow mode. Every engine process picks a change up within a few seconds.",
}

var tenantControlSetCmd = &cobra.Command{
	Use:   "set",
	Short: "set replaces a tenant's controls. A control left out is unset.",
	Run: func(cmd *cobra.Command, args []string) {
		opts, err := setOptsFromFlags(cmd)
		if err != nil {
			fatalf("%v", err)
		}

		opts.TenantId = parseTenantControlTenantID()

		withTenantControls(func(ctx context.Context, repo v1.TenantControlRepository) error {
			row, err := repo.Set(ctx, opts)
			if err != nil {
				return err
			}

			printTenantControls(row)

			return nil
		})
	},
}

var tenantControlClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "clear removes a tenant's controls. Ending a pause this way (or with set) moves the tenant's schedule deadlines forward by the length of the pause; plain SQL does not.",
	Run: func(cmd *cobra.Command, args []string) {
		tenantID := parseTenantControlTenantID()

		withTenantControls(func(ctx context.Context, repo v1.TenantControlRepository) error {
			cleared, err := repo.Clear(ctx, tenantID)
			if err != nil {
				return err
			}

			if !cleared {
				fmt.Printf("tenant %s had no controls\n", tenantID)
				return nil
			}

			fmt.Printf("cleared the controls of tenant %s\n", tenantID)

			return nil
		})
	},
}

var tenantControlListCmd = &cobra.Command{
	Use:   "list",
	Short: "list prints every tenant that has a control set.",
	Run: func(cmd *cobra.Command, args []string) {
		withTenantControls(func(ctx context.Context, repo v1.TenantControlRepository) error {
			rows, err := repo.List(ctx)
			if err != nil {
				return err
			}

			printTenantControls(rows...)

			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(tenantControlCmd)
	tenantControlCmd.AddCommand(tenantControlSetCmd, tenantControlClearCmd, tenantControlListCmd)

	for _, cmd := range []*cobra.Command{tenantControlSetCmd, tenantControlClearCmd} {
		cmd.Flags().StringVar(&tenantControlTenantID, "tenant-id", "", "id of the tenant")
		_ = cmd.MarkFlagRequired("tenant-id")
	}

	tenantControlSetCmd.Flags().BoolVar(&tenantControlPaused, "paused", false, "stop scheduling the tenant and refuse its new runs and events")
	tenantControlSetCmd.Flags().Int32Var(&tenantControlIngest, "ingest-per-second", 0, "the most new runs and events each engine or API process admits for the tenant per second, at least 1; the rest are refused with a retry delay")
	tenantControlSetCmd.Flags().Int32Var(&tenantControlDBConnPercent, "db-conn-percent", 0, "slow mode as a connection cap (not recommended: it stalls the shared message queues): the share of each process's connection pool the tenant may hold, from 1 to 100")
	tenantControlSetCmd.Flags().Int32Var(&tenantControlDBTimePercent, "db-time-percent", 0, "database time budget: the share of each process's pool connection-seconds the tenant may hold on average, from 1 to 100")
	tenantControlSetCmd.Flags().StringVar(&tenantControlReason, "reason", "", "why, for the next operator")
	_ = tenantControlSetCmd.MarkFlagRequired("reason")
}

// setOptsFromFlags checks set's flags and returns its options without the tenant id.
func setOptsFromFlags(cmd *cobra.Command) (v1.SetTenantControlOpts, error) {
	opts := v1.SetTenantControlOpts{
		Paused: tenantControlPaused,
		Reason: tenantControlReason,
	}

	var err error

	if opts.IngestPerSecond, err = intFlag(cmd, "ingest-per-second", tenantControlIngest, math.MaxInt32); err != nil {
		return opts, err
	}

	if opts.DBTimePercent, err = intFlag(cmd, "db-time-percent", tenantControlDBTimePercent, 100); err != nil {
		return opts, err
	}

	if opts.DBConnPercent, err = intFlag(cmd, "db-conn-percent", tenantControlDBConnPercent, 100); err != nil {
		return opts, err
	}

	if !opts.Paused && opts.IngestPerSecond == nil && opts.DBTimePercent == nil && opts.DBConnPercent == nil {
		return opts, errors.New("pass --paused, --ingest-per-second, --db-time-percent, --db-conn-percent or a combination; to remove a tenant's controls use clear")
	}

	if opts.Reason == "" {
		return opts, errors.New("--reason must not be empty")
	}

	return opts, nil
}

// intFlag returns the flag's value if it was passed, after checking it is from 1 to highest.
func intFlag(cmd *cobra.Command, name string, value, highest int32) (*int32, error) {
	if !cmd.Flags().Changed(name) {
		return nil, nil
	}

	if value < 1 || value > highest {
		return nil, fmt.Errorf("--%s must be from 1 to %d, got %d", name, highest, value)
	}

	return &value, nil
}

func parseTenantControlTenantID() uuid.UUID {
	tenantID, err := uuid.Parse(tenantControlTenantID)
	if err != nil {
		fatalf("--tenant-id is not a valid uuid: %v", err)
	}

	return tenantID
}

func withTenantControls(fn func(ctx context.Context, repo v1.TenantControlRepository) error) {
	dc, err := loader.NewConfigLoader(configDirectory).InitDataLayer()
	if err != nil {
		fatalf("could not connect to the database: %v", err)
	}

	defer dc.Disconnect() // nolint: errcheck

	// ending a pause of a large backlog can take up to the resume statement timeout
	ctx, cancel := context.WithTimeout(context.Background(), v1.TenantControlResumeTimeout+time.Minute)
	defer cancel()

	if err := fn(ctx, dc.V1.TenantControl()); err != nil {
		fatalf("%v", err)
	}
}

func printTenantControls(rows ...*sqlcv1.V1TenantControl) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TENANT ID\tPAUSED SINCE\tINGEST PER SECOND\tDB TIME PERCENT\tDB CONN PERCENT\tUPDATED AT\tREASON")

	for _, row := range rows {
		paused := "-"
		if row.Paused {
			paused = "yes"
			if row.PausedAt.Valid {
				paused = row.PausedAt.Time.UTC().Format(time.RFC3339)
			}
		}

		ingest := "-"
		if row.IngestPerSecond.Valid {
			ingest = strconv.Itoa(int(row.IngestPerSecond.Int32))
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", row.TenantID, paused, ingest, describePercent(row.DbTimePercent), describePercent(row.DbConnPercent), row.UpdatedAt.Time.UTC().Format(time.RFC3339), row.Reason)
	}

	_ = w.Flush()
}

func describePercent(percent pgtype.Int4) string {
	if !percent.Valid {
		return "default"
	}

	return strconv.Itoa(int(percent.Int32))
}

func fatalf(format string, args ...any) {
	log.Printf("Fatal: "+format, args...)
	os.Exit(1)
}

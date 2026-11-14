package cli

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

var (
	tenantControlTenantID string
	tenantControlIngest   int32
	tenantControlReason   string
)

var tenantControlCmd = &cobra.Command{
	Use:   "tenant-control",
	Short: "command for operator controls on a tenant. Every engine and API process applies a change within a few seconds.",
}

var tenantControlSetCmd = &cobra.Command{
	Use:   "set",
	Short: "set replaces a tenant's controls.",
	Run: func(cmd *cobra.Command, args []string) {
		opts, err := tenantControlSetOpts(tenantControlTenantID, tenantControlIngest, tenantControlReason)
		if err != nil {
			fatalf("%v", err)
		}

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
	Short: "clear removes a tenant's controls.",
	Run: func(cmd *cobra.Command, args []string) {
		tenantID, err := uuid.Parse(tenantControlTenantID)
		if err != nil {
			fatalf("--tenant-id is not a valid uuid: %v", err)
		}

		withTenantControls(func(ctx context.Context, repo v1.TenantControlRepository) error {
			cleared, err := repo.Clear(ctx, tenantID)
			if err != nil {
				return err
			}

			if cleared {
				fmt.Printf("cleared the controls of tenant %s\n", tenantID)
			} else {
				fmt.Printf("tenant %s had no controls\n", tenantID)
			}

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

	tenantControlSetCmd.Flags().Int32Var(&tenantControlIngest, "ingest-per-second", 0, "the most new runs and events each engine or API process admits for the tenant per second, at least 1; the rest are refused with a retry delay. The limit is per process, so with N processes serving the tenant it admits up to N times this")
	tenantControlSetCmd.Flags().StringVar(&tenantControlReason, "reason", "", "why, for the next operator")
	_ = tenantControlSetCmd.MarkFlagRequired("ingest-per-second")
	_ = tenantControlSetCmd.MarkFlagRequired("reason")
}

func tenantControlSetOpts(tenantID string, ingestPerSecond int32, reason string) (v1.SetTenantControlOpts, error) {
	id, err := uuid.Parse(tenantID)
	if err != nil {
		return v1.SetTenantControlOpts{}, fmt.Errorf("--tenant-id is not a valid uuid: %w", err)
	}

	if ingestPerSecond < 1 {
		return v1.SetTenantControlOpts{}, fmt.Errorf("--ingest-per-second must be at least 1, got %d", ingestPerSecond)
	}

	if reason == "" {
		return v1.SetTenantControlOpts{}, errors.New("--reason must not be empty")
	}

	return v1.SetTenantControlOpts{TenantId: id, IngestPerSecond: ingestPerSecond, Reason: reason}, nil
}

// withTenantControls runs fn on a data layer of its own and exits the process if either fails.
func withTenantControls(fn func(ctx context.Context, repo v1.TenantControlRepository) error) {
	if err := runWithTenantControls(fn); err != nil {
		fatalf("%v", err)
	}
}

func runWithTenantControls(fn func(ctx context.Context, repo v1.TenantControlRepository) error) error {
	dc, err := loader.NewConfigLoader(configDirectory).InitDataLayer()
	if err != nil {
		return fmt.Errorf("could not connect to the database: %w", err)
	}

	defer dc.Disconnect() // nolint: errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return fn(ctx, dc.V1.TenantControl())
}

func printTenantControls(rows ...*sqlcv1.V1TenantControl) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TENANT ID\tINGEST PER SECOND\tUPDATED AT\tREASON")

	for _, row := range rows {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", row.TenantID, row.IngestPerSecond, row.UpdatedAt.Time.UTC().Format(time.RFC3339), row.Reason)
	}

	_ = w.Flush()
}

func fatalf(format string, args ...any) {
	log.Printf("Fatal: "+format, args...)
	os.Exit(1)
}

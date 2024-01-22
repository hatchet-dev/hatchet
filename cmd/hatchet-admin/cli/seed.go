package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/internal/config/loader"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

// seedCmd seeds the database with initial data
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "seed create initial data in the database.",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		configLoader := loader.NewConfigLoader(configDirectory)
		err = runSeed(configLoader)

		if err != nil {
			fmt.Printf("Fatal: could not run seed command: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
}

func runSeed(cf *loader.ConfigLoader) error {
	// load the config
	dc, err := cf.LoadDatabaseConfig()

	if err != nil {
		panic(err)
	}

	shouldSeedUser := dc.Seed.AdminEmail != "" && dc.Seed.AdminPassword != ""
	var userId string

	if shouldSeedUser {
		// seed an example user
		hashedPw, err := repository.HashPassword(dc.Seed.AdminPassword)

		if err != nil {
			return err
		}

		user, err := dc.Repository.User().GetUserByEmail(dc.Seed.AdminEmail)

		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				user, err = dc.Repository.User().CreateUser(&repository.CreateUserOpts{
					Email:         dc.Seed.AdminEmail,
					Name:          repository.StringPtr(dc.Seed.AdminName),
					EmailVerified: repository.BoolPtr(true),
					Password:      hashedPw,
				})

				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		userId = user.ID
	}

	tenant, err := dc.Repository.Tenant().GetTenantBySlug("default")

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// seed an example tenant
			// initialize a tenant
			tenant, err = dc.Repository.Tenant().CreateTenant(&repository.CreateTenantOpts{
				ID:   &dc.Seed.DefaultTenantID,
				Name: dc.Seed.DefaultTenantName,
				Slug: dc.Seed.DefaultTenantSlug,
			})

			if err != nil {
				return err
			}

			fmt.Println("created tenant", tenant.ID)

			// add the user to the tenant
			_, err = dc.Repository.Tenant().CreateTenantMember(tenant.ID, &repository.CreateTenantMemberOpts{
				Role:   "OWNER",
				UserId: userId,
			})

			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if dc.Seed.IsDevelopment {
		err = seedDev(dc.Repository, tenant.ID)

		if err != nil {
			return err
		}
	}

	return nil
}

func seedDev(repo repository.Repository, tenantId string) error {
	_, err := repo.Workflow().GetWorkflowByName(tenantId, "test-workflow")

	if err != nil {
		if !errors.Is(err, db.ErrNotFound) {
			return err
		}

		wf, err := repo.Workflow().CreateNewWorkflow(tenantId, &repository.CreateWorkflowVersionOpts{
			Name:        "test-workflow",
			Description: repository.StringPtr("This is a test workflow."),
			Version:     "v0.1.0",
			EventTriggers: []string{
				"user:create",
			},
			Tags: []repository.CreateWorkflowTagOpts{
				{
					Name: "Preview",
				},
			},
			Jobs: []repository.CreateWorkflowJobOpts{
				{
					Name: "job-name",
					Steps: []repository.CreateWorkflowStepOpts{
						{
							ReadableId: "echo1",
							Action:     "echo:echo",
						},
						{
							ReadableId: "echo2",
							Action:     "echo:echo",
							Parents: []string{
								"echo1",
							},
						},
						{
							ReadableId: "echo3",
							Action:     "echo:echo",
							Parents: []string{
								"echo2",
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		fmt.Println("created workflow", wf.ID, wf.Workflow().Name)
	}

	return nil
}

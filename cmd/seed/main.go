package main

import (
	"errors"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/config/loader"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func main() {
	// init the repository
	cf := &loader.ConfigLoader{}

	// load the config
	dc, err := cf.LoadDatabaseConfig()

	if err != nil {
		panic(err)
	}

	// seed an example user
	hashedPw, err := repository.HashPassword("Admin123!!")

	if err != nil {
		panic(err)
	}

	user, err := dc.Repository.User().GetUserByEmail("admin@example.com")

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			user, err = dc.Repository.User().CreateUser(&repository.CreateUserOpts{
				Email:         "admin@example.com",
				Name:          repository.StringPtr("Admin"),
				EmailVerified: repository.BoolPtr(true),
				Password:      *hashedPw,
			})

			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	tenant, err := dc.Repository.Tenant().GetTenantBySlug("default")

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// seed an example tenant
			// initialize a tenant
			tenant, err = dc.Repository.Tenant().CreateTenant(&repository.CreateTenantOpts{
				Name: "Default",
				Slug: "default",
			})

			if err != nil {
				panic(err)
			}

			fmt.Println("created tenant", tenant.ID)

			// add the user to the tenant
			_, err = dc.Repository.Tenant().CreateTenantMember(tenant.ID, &repository.CreateTenantMemberOpts{
				Role:   "OWNER",
				UserId: user.ID,
			})

			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	// seed example workflows
	firstInput, _ := datautils.ToJSONType(map[string]interface{}{
		"message": "Username is {{ .input.username }}",
	})

	secondInput, _ := datautils.ToJSONType(map[string]interface{}{
		"message": "Above message is: {{ .steps.echo1.message }}",
	})

	thirdInput, _ := datautils.ToJSONType(map[string]interface{}{
		"message": "Above message is: {{ .steps.echo1.message }}",
	})

	_, err = dc.Repository.Workflow().CreateNewWorkflow(tenant.ID, &repository.CreateWorkflowVersionOpts{
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
						Inputs:     firstInput,
					},
					{
						ReadableId: "echo2",
						Action:     "echo:echo",
						Inputs:     secondInput,
					},
					{
						ReadableId: "echo3",
						Action:     "echo:echo",
						Inputs:     thirdInput,
					},
				},
			},
		},
	})

	if err != nil {
		panic(err)
	}

	workflows, err := dc.Repository.Workflow().ListWorkflowsForEvent(tenant.ID, "user:create")

	if err != nil {
		panic(err)
	}

	for _, workflow := range workflows {
		fmt.Println("created workflow", workflow.ID, workflow.Workflow().Name, workflow.Version)
	}

	// seed example events
	generateEvents(dc.Repository, tenant.ID)
}

func generateEvents(repo repository.Repository, tenantId string) {
	for i := 0; i < 600; i++ {
		_, err := repo.Event().CreateEvent(&repository.CreateEventOpts{
			TenantId: tenantId,
			Key:      fmt.Sprintf("user-%d:create", i),
		})

		if err != nil {
			panic(err)
		}
	}
}

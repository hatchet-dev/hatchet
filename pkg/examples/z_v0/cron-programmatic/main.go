package main

import (
	"context"
	"fmt"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// > Create
// ... normal workflow definition
type printOutput struct{}

func print(ctx context.Context) (result *printOutput, err error) {
	fmt.Println("called print:print")

	return &printOutput{}, nil
}

// ,
func main() {
	// ... initialize client, worker and workflow
	err := godotenv.Load()

	if err != nil {
		panic(err)
	}

	c, err := client.New()

	if err != nil {
		panic(err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)

	if err != nil {
		panic(err)
	}

	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.NoTrigger(),
			Name:        "cron-workflow",
			Description: "Demonstrates a simple cron workflow",
			Steps: []*worker.WorkflowStep{
				worker.Fn(print),
			},
		},
	)

	if err != nil {
		panic(err)
	}

	interrupt := cmdutils.InterruptChan()

	cleanup, err := w.Start()

	if err != nil {
		panic(err)
	}

	// ,

	go func() {
		// 👀 define the cron expression to run every minute
		cron, err := c.Cron().Create(
			context.Background(),
			"cron-workflow",
			&client.CronOpts{
				Name:       "every-minute",
				Expression: "* * * * *",
				Input: map[string]interface{}{
					"message": "Hello, world!",
				},
				AdditionalMetadata: map[string]string{},
			},
		)

		if err != nil {
			panic(err)
		}

		fmt.Println(*cron.Name, cron.Cron)
	}()

	// ... wait for interrupt signal

	<-interrupt

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}

	// ,
}

// !!

func ListCrons() {

	c, err := client.New()

	if err != nil {
		panic(err)
	}

	// > List
	crons, err := c.Cron().List(context.Background())
	// !!

	if err != nil {
		panic(err)
	}

	for _, cron := range *crons.Rows {
		fmt.Println(cron.Cron, *cron.Name)
	}
}

func DeleteCron(id string) {
	c, err := client.New()

	if err != nil {
		panic(err)
	}

	// > Delete
	// 👀 id is the cron's metadata id, can get it via cron.Metadata.Id
	err = c.Cron().Delete(context.Background(), id)
	// !!

	if err != nil {
		panic(err)
	}

}

package main

import (
	_ "embed"
	"os"
	"strings"

	"context"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/integrations/slack"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type userCreateEvent struct {
	Username string `json:"username"`
}

type actionInput struct {
	Message string `json:"message"`
}

//go:embed .hatchet/slack-channel.yaml
var SlackChannelWorkflow []byte

func init() {
	// initialize the slack channel workflow with SLACK_USER_ID
	slackUserId := os.Getenv("SLACK_USER_ID")

	if slackUserId == "" {
		panic("SLACK_USER_ID environment variable must be set")
	}

	slackFileWithReplacedEnv := strings.Replace(string(SlackChannelWorkflow), "$SLACK_USER_ID", slackUserId, 1)

	SlackChannelWorkflow = []byte(slackFileWithReplacedEnv)
}

func main() {
	// read the slack workflow
	slackWorkflowFile, err := types.ParseYAML(context.Background(), SlackChannelWorkflow)

	if err != nil {
		panic(err)
	}

	// render the slack workflow using the environment variable SLACK_USER_ID
	slackToken := os.Getenv("SLACK_TOKEN")
	slackTeamId := os.Getenv("SLACK_TEAM_ID")

	if slackToken == "" {
		panic("SLACK_TOKEN environment variable must be set")
	}

	if slackTeamId == "" {
		panic("SLACK_TEAM_ID environment variable must be set")
	}

	slackInt := slack.NewSlackIntegration(slackToken, slackTeamId, true)

	client, err := client.New(
		client.InitWorkflows(),
		client.WithWorkflows([]*types.Workflow{
			&slackWorkflowFile,
		}),
	)

	if err != nil {
		panic(err)
	}

	// Create a worker. This automatically reads in a TemporalClient from .env and workflow files from the .hatchet
	// directory, but this can be customized with the `worker.WithTemporalClient` and `worker.WithWorkflowFiles` options.
	worker, err := worker.NewWorker(
		worker.WithDispatcherClient(
			client.Dispatcher(),
		),
		worker.WithIntegration(
			slackInt,
		),
	)

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContext(cmdutils.InterruptChan())
	defer cancel()

	go worker.Start(interruptCtx)

	testEvent := userCreateEvent{
		Username: "testing1233344",
	}

	// push an event
	err = client.Event().Push(
		context.Background(),
		"user:create",
		testEvent,
	)

	if err != nil {
		panic(err)
	}

	for {
		select {
		case <-interruptCtx.Done():
			return
		default:
			time.Sleep(time.Second)
		}
	}
}

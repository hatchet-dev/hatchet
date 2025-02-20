package main

import (
	"context"
	_ "embed"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/integrations/slack"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type teamCreateEvent struct {
	Name string `json:"name"`
}

//go:embed .hatchet/slack-channel.yaml
var SlackChannelWorkflow []byte

func init() {
	err := godotenv.Load()

	if err != nil {
		panic(err)
	}

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

	worker, err := worker.NewWorker(
		worker.WithClient(
			client,
		),
		worker.WithIntegration(
			slackInt,
		),
	)

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	go worker.Start()

	testEvent := teamCreateEvent{
		Name: "test-team-2",
	}

	// push an event
	err = client.Event().Push(
		context.Background(),
		"team:create",
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

package main

import (
	"fmt"
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Stripe webhook task
	type StripePaymentInput struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				Customer string `json:"customer"`
				Amount   int    `json:"amount"`
			} `json:"object"`
		} `json:"data"`
	}

	stripePayment := client.NewStandaloneTask(
		"handle-stripe-payment",
		func(ctx hatchet.Context, input StripePaymentInput) (*struct {
			Customer string `json:"customer"`
			Amount   int    `json:"amount"`
		}, error) {
			fmt.Printf("Payment of %d from %s\n", input.Data.Object.Amount, input.Data.Object.Customer)
			return &struct {
				Customer string `json:"customer"`
				Amount   int    `json:"amount"`
			}{
				Customer: input.Data.Object.Customer,
				Amount:   input.Data.Object.Amount,
			}, nil
		},
		hatchet.WithWorkflowEvents("stripe:payment_intent.succeeded"),
	)
	// !!

	// > GitHub webhook task
	type GitHubPRInput struct {
		Action      string `json:"action"`
		PullRequest struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
		} `json:"pull_request"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}

	githubPR := client.NewStandaloneTask(
		"handle-github-pr",
		func(ctx hatchet.Context, input GitHubPRInput) (*struct {
			Repo string `json:"repo"`
			PR   int    `json:"pr"`
		}, error) {
			fmt.Printf("PR #%d opened on %s: %s\n", input.PullRequest.Number, input.Repository.FullName, input.PullRequest.Title)
			return &struct {
				Repo string `json:"repo"`
				PR   int    `json:"pr"`
			}{
				Repo: input.Repository.FullName,
				PR:   input.PullRequest.Number,
			}, nil
		},
		hatchet.WithWorkflowEvents("github:pull_request:opened"),
	)
	// !!

	// > Slack event subscription task
	type SlackEventInput struct {
		Event struct {
			Type    string `json:"type"`
			User    string `json:"user"`
			Text    string `json:"text"`
			Channel string `json:"channel"`
		} `json:"event"`
	}

	slackMention := client.NewStandaloneTask(
		"handle-slack-mention",
		func(ctx hatchet.Context, input SlackEventInput) (*struct {
			Handled bool `json:"handled"`
		}, error) {
			fmt.Printf("Mentioned by %s in %s: %s\n", input.Event.User, input.Event.Channel, input.Event.Text)
			return &struct {
				Handled bool `json:"handled"`
			}{Handled: true}, nil
		},
		hatchet.WithWorkflowEvents("slack:event:app_mention"),
	)
	// !!

	// > Slack slash command task
	type SlackCommandInput struct {
		Command     string `json:"command"`
		Text        string `json:"text"`
		UserName    string `json:"user_name"`
		ResponseURL string `json:"response_url"`
	}

	slackCommand := client.NewStandaloneTask(
		"handle-slack-command",
		func(ctx hatchet.Context, input SlackCommandInput) (*struct {
			Command string `json:"command"`
			Args    string `json:"args"`
		}, error) {
			fmt.Printf("%s ran %s %s\n", input.UserName, input.Command, input.Text)
			return &struct {
				Command string `json:"command"`
				Args    string `json:"args"`
			}{
				Command: input.Command,
				Args:    input.Text,
			}, nil
		},
		hatchet.WithWorkflowEvents("slack:command:/deploy"),
	)
	// !!

	// > Slack interaction task
	type SlackInteractionInput struct {
		Type    string `json:"type"`
		Actions []struct {
			ActionID string `json:"action_id"`
		} `json:"actions"`
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}

	slackInteraction := client.NewStandaloneTask(
		"handle-slack-interaction",
		func(ctx hatchet.Context, input SlackInteractionInput) (*struct {
			Action string `json:"action"`
		}, error) {
			action := input.Actions[0]
			fmt.Printf("%s clicked button: %s\n", input.User.Username, action.ActionID)
			return &struct {
				Action string `json:"action"`
			}{Action: action.ActionID}, nil
		},
		hatchet.WithWorkflowEvents("slack:interaction:block_actions"),
	)
	// !!

	worker, err := client.NewWorker("webhook-worker",
		hatchet.WithWorkflows(
			stripePayment,
			githubPR,
			slackMention,
			slackCommand,
			slackInteraction,
		),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}

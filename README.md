[![slack](https://img.shields.io/badge/Join%20Our%20Community-Slack-blue)](https://join.slack.com/t/hatchet-co/signup) [![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT)

<!-- [![Go Reference](https://pkg.go.dev/badge/github.com/hatchet-dev/hatchet.svg)](https://pkg.go.dev/github.com/hatchet-dev/hatchet) -->

## Introduction

_**Note:** Hatchet is in early development. Changes are not guaranteed to be backwards-compatible. If you'd like to run Hatchet in production, feel free to reach out on Slack for tips._

Hatchet is an event storage API and workflow engine for distributed applications. Using Hatchet, you can create workers which process a set of background tasks based on different triggers, like events created within your system or a cron schedule.

As a simple example, let's say you want to perform 3 actions when a user has signed up for your app:

1. Initialize a set of resources for the user (perhaps a sandbox environment for testing).
2. Send the user an automated greeting over email
3. Add the user to a newsletter campaign

With Hatchet, this would look something like the following:

```yaml
name: "post-user-sign-up"
version: v0.2.0
triggers:
  events:
    - user:create
jobs:
  create-resources:
    steps:
      - id: createSandbox
        action: sandbox:create
        timeout: 60s
  greet-user:
    steps:
      - id: greetUser
        action: postmark:email-from-template
        timeout: 15s
        with:
          firstName: "{{ .user.firstName }}"
          email: "{{ .user.email }}"
  add-to-newsletter:
    steps:
      - id: addUserToNewsletter
        action: newsletter:add-user
        timeout: 15s
        with:
          email: "{{ .user.email }}"
```

In your codebase, you would then create a worker which could perform the following actions:

- `sandbox:create` responsible for creating/tearing down a sandbox environment
- `postmark:email-from-template` for sending an email from a template
- `newsletter:add-user` for adding a user to a newsletter campaign

Ultimately, the goal of Hatchet workflows are that you don't need to write these actions yourself -- creating a robust set of prebuilt integrations is one of the goals of the project.

### Why is this useful?

- When deploying a workflow engine or task queue, one of the first breaking points is requiring a robust event architecture for monitoring and replaying events. Hatchet provides this out of the box, allowing you to replay your events and retrigger your workflows.
- No need to build all of your plumbing logic (action 1 -> event 1 -> action 2 -> event 2). Just define your jobs and steps and write your business logic. This is particularly useful the more complex your workflows become.
- Using prebuilt integrations with a standard interface makes building auxiliary services like notification systems, billing, backups, and auditing much easier. **Please file an issue if you'd like to see an integration supported.** The following are on the roadmap:
  - Email providers: Sendgrid, Postmark, AWS SES
  - Stripe
  - AWS S3
- Additionally, if you're already familiar with/using a workflow engine, making workflows declarative provides several benefits:
  - Makes spec'ing, debugging and visualizing workflows much simpler
  - Automatically updates triggers, schedules, and timeouts when they change, rather than doing this through a UI/CLI/SDK
  - Makes monitoring easier to build by logically separating units of work - jobs will automatically correspond to `BeginSpan`. OpenTelemetry support is on the roadmap.

## Getting Started

For a set of end-to-end examples, see the [examples](./examples) directory.

### Starting Hatchet

We are working on making it easier to start a Hatchet server. For now, see the [contributing guide](./CONTRIBUTING.md) for starting the Hatchet engine.

### Writing a Workflow

By default, Hatchet searches for workflows in the `.hatchet` folder relative to the directory you run your application in.

There are two main sections of a workflow file:

**Triggers (using `on`)**

This section specifies what triggers a workflow. This can be events or a crontab-like schedule. For example, the following are valid triggers:

```yaml
on:
  - eventkey1
  - eventkey2
```

```yaml
on:
  cron:
    schedule: "*/15 * * * *"
```

**Jobs**

After defining your triggers, you define a list of jobs to run based on the triggers. **Jobs run in parallel.** Jobs contain the following fields:

```yaml
# ...
jobs:
  my-awesome-job:
    # (optional) A timeout value for the entire job
    timeout: 60s
    # (required) A set of steps for the job; see below
    steps: []
```

Within each job, there are a set of **steps** which run sequentially. A step can contain the following fields:

```yaml
# (required) the name of the step
name: Step 1
# (required) a unique id for the step (can be referenced by future steps)
id: step-1
# (required) the action id in the form of "integration_id:action".
action: "slack:create-channel"
# (required) the timeout of the individual step
timeout: 15s
# (optional or required, depending on integration) input data to the integration
with:
  key: val
```

### Creating a Worker

Workers can be created using:

```go
package main

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserId   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type actionInput struct {
	Message string `json:"message"`
}

func main() {
	client, err := client.New(
		client.InitWorkflows(),
	)

	if err != nil {
		panic(err)
	}

	worker, err := worker.NewWorker(
		worker.WithDispatcherClient(
			client.Dispatcher(),
		),
	)

	if err != nil {
		panic(err)
	}

	err = worker.RegisterAction("echo:echo", func(ctx context.Context, input *actionInput) (result any, err error) {
		return map[string]interface{}{
			"message": input.Message,
		}, nil
	})

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContext(cmdutils.InterruptChan())
	defer cancel()

  worker.Start(interruptCtx)
}
```

You can configure the worker with your own set of workflow files using the `client.WithWorkflowFiles` option.

### Triggering Events

To trigger events from your main application, use the `client.Event().Push` method:

```go
package main

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserId   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

func main() {
	client, err := client.New()

  testEvent := userCreateEvent{
		Username: "echo-test",
		UserId:   "1234",
		Data: map[string]string{
			"test": "test",
		},
	}

	err = client.Event().Push(
		context.Background(),
		"user:create",
		testEvent,
	)

	if err != nil {
		panic(err)
	}
}
```

You can configure the dispatcher with your own set of workflow files using the `dispatcher.WithWorkflowFiles` option.

## Why should I care?

**If you're unfamiliar with background task processing**

Many APIs start out without a task processing/worker service. You might not need it, but at a certain level of complexity, you probably will. There are a few use-cases where workers start to make sense:

1. You need to run scheduled tasks which that aren't triggered from your core API. For example, this may be a daily cleanup task, like traversing soft-deleted database entries or backing up data to S3.
2. You need to run tasks which are triggered by API events, but aren't required for the core business logic of the handler. For example, you want to add a user to your CRM after they sign up.

For both of these cases, it's typical to re-use a lot of core functionality from your API, so the most natural place to start is by adding some automation within your API itself; for example, after returning `201 Created`, you might send a greeting to the user, initialize a sandbox environment, send an internal notification that a user signed up, etc, all within your API handlers. Let's say you've handled this case as following:

```go
// Hypothetical handler called via a routing package, let's just pretend it returns an error
func MyHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    // Boilerplate code to parse the request
    var newUser User
    err := json.NewDecoder(r.Body).Decode(&newUser)
    if err != nil {
      http.Error(w, "Invalid user data", http.StatusBadRequest)
      return err
    }

    // Validate email and password fields...
    // (Add your validation logic here)

    // Create a user in the database
    user, err := createUser(ctx, newUser.Email, newUser.Password)
    if err != nil {
      // Handle database errors, such as unique constraint violation
      http.Error(w, "Error creating user", http.StatusInternalServerError)
      return err
    }

    // Return 201 created with user type
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)

    // Send user a greeting
    err := email.SendGreetingEmail(context.Background(), user)

    if err != nil {
      // can't return an error, since header is already set
      fmt.Println(err)
    }

    // ... other post-signup operations
}
```

At some point, you realize all of these background operations don't really belong in the handler -- when they're part of the handler, they're more difficult to monitor and observe, difficult to retry (especially if a third-party service goes down), and bloat your handlers (which could cause goroutine leakage or memory issues).

This is where a service like Hatchet suited for background/task processing comes in.

## I'd Like to Contribute

Hatchet is still in very early development -- as a result, there are very few development docs. However, please feel free to reach out on the #contributing channel on [Slack](https://join.slack.com/t/hatchet-co/signup) to shape the direction of the project.

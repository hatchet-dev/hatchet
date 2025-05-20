<div align="center">
<a href ="https://hatchet.run?utm_source=github&utm_campaign=readme">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./frontend/docs/public/hatchet_logo.png">
  <img width="200" alt="Hatchet Logo" src="./frontend/docs/public/hatchet_logo_light.png">
</picture>
</a>

### Run Background Tasks at Scale

[![Docs](https://img.shields.io/badge/docs-docs.hatchet.run-3F16E4)](https://docs.hatchet.run) [![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT) [![Go Reference](https://pkg.go.dev/badge/github.com/hatchet-dev/hatchet.svg)](https://pkg.go.dev/github.com/hatchet-dev/hatchet) [![NPM Downloads](https://img.shields.io/npm/dm/%40hatchet-dev%2Ftypescript-sdk)](https://www.npmjs.com/package/@hatchet-dev/typescript-sdk)

[![Discord](https://img.shields.io/discord/1088927970518909068?style=social&logo=discord)](https://hatchet.run/discord)
[![Twitter](https://img.shields.io/twitter/url/https/twitter.com/hatchet-dev.svg?style=social&label=Follow%20%40hatchet-dev)](https://twitter.com/hatchet_dev)
[![GitHub Repo stars](https://img.shields.io/github/stars/hatchet-dev/hatchet?style=social)](https://github.com/hatchet-dev/hatchet)

  <p align="center">
    <a href="https://cloud.onhatchet.run">Hatchet Cloud</a>
    ·
    <a href="https://docs.hatchet.run">Documentation</a>
    ·
    <a href="https://hatchet.run">Website</a>
    ·
    <a href="https://github.com/hatchet-dev/hatchet/issues">Issues</a>
  </p>

</div>

### What is Hatchet?

Hatchet is a platform for running background tasks, built on top of Postgres. Instead of managing your own task queue or pub/sub system, you can use Hatchet to distribute your functions between a set of workers with minimal configuration or infrastructure.

### When should I use Hatchet?

Background tasks are critical for offloading work from your main web application. Usually background tasks are sent through a FIFO (first-in-first-out) queue, which helps guard against traffic spikes (queues can absorb a lot of load) and ensures that tasks are retried when your task handlers error out. Most stacks begin with a library-based queue backed by Redis or RabbitMQ (like Celery or BullMQ). But as your tasks become more complex, these queues become difficult to debug, monitor and start to fail in unexpected ways.

This is where Hatchet comes in. Hatchet is a full-featured background task management platform, with built-in support for chaining complex tasks together into workflows, alerting on failures, making tasks more durable, and viewing tasks in a real-time web dashboard.

### Features

<details open><summary><strong>📥 Queues</strong></summary>

####

Hatchet is built on a durable task queue that enqueues your tasks and sends them to your workers at a rate that your workers can handle. Hatchet will track the progress of your task and ensure that the work gets completed (or you get alerted), even if your application crashes.

**This is particularly useful for:**

- Ensuring that you never drop a user request
- Flattening large spikes in your application
- Breaking large, complex logic into smaller, reusable tasks

[Read more ➶](https://docs.hatchet.run/home/your-first-task)

- <details>

    <summary><code>Python</code></summary>

  ```python
  # 1. Define your task input
  class SimpleInput(BaseModel):
      message: str

  # 2. Define your task using hatchet.task
  @hatchet.task(name="SimpleWorkflow", input_validator=SimpleInput)
  def simple(input: SimpleInput, ctx: Context) -> dict[str, str]:
      return {
        "transformed_message": input.message.lower(),
      }

  # 3. Register your task on your worker
  worker = hatchet.worker("test-worker", workflows=[simple])
  worker.start()

  # 4. Invoke tasks from your application
  simple.run(SimpleInput(message="Hello World!"))
  ```

  </details>

- <details>

    <summary><code>Typescript</code></summary>

  ```ts
  // 1. Define your task input
  export type SimpleInput = {
    Message: string;
  };

  // 2. Define your task using hatchet.task
  export const simple = hatchet.task({
    name: "simple",
    fn: (input: SimpleInput) => {
      return {
        TransformedMessage: input.Message.toLowerCase(),
      };
    },
  });

  // 3. Register your task on your worker
  const worker = await hatchet.worker("simple-worker", {
    workflows: [simple],
  });

  await worker.start();

  // 4. Invoke tasks from your application
  await simple.run({
    Message: "Hello World!",
  });
  ```

  </details>

- <details>

    <summary><code>Go</code></summary>

  ```go
  // 1. Define your task input
  type SimpleInput struct {
    Message string `json:"message"`
  }

  // 2. Define your task using factory.NewTask
  simple := factory.NewTask(
    create.StandaloneTask{
      Name: "simple-task",
    }, func(ctx worker.HatchetContext, input SimpleInput) (*SimpleResult, error) {
      return &SimpleResult{
        TransformedMessage: strings.ToLower(input.Message),
      }, nil
    },
    hatchet,
  )

  // 3. Register your task on your worker
  worker, err := hatchet.Worker(v1worker.WorkerOpts{
    Name: "simple-worker",
    Workflows: []workflow.WorkflowBase{
      simple,
    },
  })

  worker.StartBlocking()

  // 4. Invoke tasks from your application
  simple.Run(context.Background(), SimpleInput{Message: "Hello, World!"})
  ```

  </details>

</details>
<details><summary><strong>🎻 Task Orchestration</strong></summary>

####

Hatchet allows you to build complex workflows that can be composed of multiple tasks. For example, if you'd like to break a workload into smaller tasks, you can use Hatchet to create a fanout workflow that spawns multiple tasks in parallel.

Hatchet supports the following mechanisms for task orchestration:

- **DAGs (directed acyclic graphs)** — pre-define the shape of your work, automatically routing the outputs of a parent task to the input of a child task. [Read more ➶](https://docs.hatchet.run/home/dags)

- **Durable tasks** — these tasks are responsible for orchestrating other tasks. They store a full history of all spawned tasks, allowing you to cache intermediate results. [Read more ➶](https://docs.hatchet.run/home/durable-execution)

- <details>

    <summary><code>Python</code></summary>

  ```python
  # 1. Define a workflow (a workflow is a collection of tasks)
  simple = hatchet.workflow(name="SimpleWorkflow")

  # 2. Attach the first task to the workflow
  @simple.task()
  def task_1(input: EmptyModel, ctx: Context) -> dict[str, str]:
      print("executed task_1")
      return {"result": "task_1"}

  # 3. Attach the second task to the workflow, which executes after task_1
  @simple.task(parents=[task_1])
  def task_2(input: EmptyModel, ctx: Context) -> None:
      first_result = ctx.task_output(task_1)
      print(first_result)

  # 4. Invoke workflows from your application
  result = simple.run(input_data)
  ```

  </details>

- <details>

    <summary><code>Typescript</code></summary>

  ```ts
  // 1. Define a workflow (a workflow is a collection of tasks)
  const simple = hatchet.workflow<DagInput, DagOutput>({
    name: "simple",
  });

  // 2. Attach the first task to the workflow
  const task1 = simple.task({
    name: "task-1",
    fn: (input) => {
      return {
        result: "task-1",
      };
    },
  });

  // 3. Attach the second task to the workflow, which executes after task-1
  const task2 = simple.task({
    name: "task-2",
    parents: [task1],
    fn: (input, ctx) => {
      const firstResult = ctx.getParentOutput(task1);
      console.log(firstResult);
    },
  });

  // 4. Invoke workflows from your application
  await simple.run({ Message: "Hello World" });
  ```

  </details>

- <details>

    <summary><code>Go</code></summary>

  ```go
  // 1. Define a workflow (a workflow is a collection of tasks)
  simple := v1.WorkflowFactory[DagInput, DagOutput](
      workflow.CreateOpts[DagInput]{
          Name: "simple-workflow",
      },
      hatchet,
  )

  // 2. Attach the first task to the workflow
  const task1 = simple.Task(
      task.CreateOpts[DagInput]{
          Name: "task-1",
          Fn: func(ctx worker.HatchetContext, _ DagInput) (*SimpleOutput, error) {
              return &SimpleOutput{
                  Result: "task-1",
              }, nil
          },
      },
  );

  // 3. Attach the second task to the workflow, which executes after task-1
  const task2 = simple.Task(
      task.CreateOpts[DagInput]{
          Name: "task-2",
          Parents: []task.NamedTask{
              step1,
          },
          Fn: func(ctx worker.HatchetContext, _ DagInput) (*SimpleOutput, error) {
              return &SimpleOutput{
                  Result: "task-2",
              }, nil
          },
      },
  );

  // 4. Invoke workflows from your application
  simple.Run(ctx, DagInput{})
  ```

  </details>

</details>
<details><summary><strong>🚦 Flow Control</strong></summary>

####

Don't let busy users crash your application. With Hatchet, you can throttle execution on a per-user, per-tenant and per-queue basis, increasing system stability and limiting the impact of busy users on the rest of your system.

Hatchet supports the following flow control primitives:

- **Concurrency** — set a concurrency limit based on a dynamic concurrency key (e.g., each user can only run 10 batch jobs at a given time). [Read more ➶](https://docs.hatchet.run/home/concurrency)

- **Rate limiting** — create both global and dynamic rate limits. [Read more ➶](https://docs.hatchet.run/home/rate-limits)

- <details>

    <summary><code>Python</code></summary>

  ```python
  # limit concurrency on a per-user basis
  flow_control_workflow = hatchet.workflow(
    name="FlowControlWorkflow",
    concurrency=ConcurrencyExpression(
      expression="input.user_id",
      max_runs=5,
      limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
    input_validator=FlowControlInput,
  )

  # rate limit a task per user to 10 tasks per minute, with each task consuming 1 unit
  @flow_control_workflow.task(
      rate_limits=[
          RateLimit(
              dynamic_key="input.user_id",
              units=1,
              limit=10,
              duration=RateLimitDuration.MINUTE,
          )
      ]
  )
  def rate_limit_task(input: FlowControlInput, ctx: Context) -> None:
      print("executed rate_limit_task")
  ```

  </details>

- <details>

    <summary><code>Typescript</code></summary>

  ```ts
  // limit concurrency on a per-user basis
  flowControlWorkflow = hatchet.workflow<SimpleInput, SimpleOutput>({
    name: "ConcurrencyLimitWorkflow",
    concurrency: {
      expression: "input.userId",
      maxRuns: 5,
      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    },
  });

  // rate limit a task per user to 10 tasks per minute, with each task consuming 1 unit
  flowControlWorkflow.task({
    name: "rate-limit-task",
    rateLimits: [
      {
        dynamicKey: "input.userId",
        units: 1,
        limit: 10,
        duration: RateLimitDuration.MINUTE,
      },
    ],
    fn: async (input) => {
      return {
        Completed: true,
      };
    },
  });
  ```

  </details>

- <details>

    <summary><code>Go</code></summary>

  ```go
  // limit concurrency on a per-user basis
  flowControlWorkflow := factory.NewWorkflow[DagInput, DagResult](
    create.WorkflowCreateOpts[DagInput]{
      Name: "simple-dag",
      Concurrency: []*types.Concurrency{
        {
          Expression:    "input.userId",
          MaxRuns:       1,
          LimitStrategy: types.GroupRoundRobin,
        },
      },
    },
    hatchet,
  )

  // rate limit a task per user to 10 tasks per minute, with each task consuming 1 unit
  flowControlWorkflow.Task(
    create.WorkflowTask[FlowControlInput, FlowControlOutput]{
      Name: "rate-limit-task",
      RateLimits: []*types.RateLimit{
        {
          Key:            "user-rate-limit",
          KeyExpr:        "input.userId",
          Units:          1,
          LimitValueExpr: 10,
          Duration:       types.Minute,
        },
      },
    }, func(ctx worker.HatchetContext, input FlowControlInput) (interface{}, error) {
      return &SimpleOutput{
        Step: 1,
      }, nil
    },
  )
  ```

  </details>

</details>
<details><summary><strong>📅 Scheduling</strong></summary>

####

Hatchet has full support for scheduling features, including cron, one-time scheduling, and pausing execution for a time duration. This is particularly useful for:

- **Cron schedules** – run data pipelines, batch processes, or notification systems on a cron schedule [Read more ➶](https://docs.hatchet.run/home/cron-runs)
- **One-time tasks** – schedule a workflow for a specific time in the future [Read more ➶](https://docs.hatchet.run/home/scheduled-runs)
- **Durable sleep** – pause execution of a task for a specific duration [Read more ➶](https://docs.hatchet.run/home/durable-execution)

- <details>

    <summary><code>Python</code></summary>

  ```python
  tomorrow = datetime.today() + timedelta(days=1)

  # schedule a task to run tomorrow
  scheduled = simple.schedule(
    tomorrow,
    SimpleInput(message="Hello, World!")
  )

  # schedule a task to run every day at midnight
  cron = simple.cron(
    "every-day",
    "0 0 * * *",
    SimpleInput(message="Hello, World!")
  )
  ```

  </details>

- <details>

    <summary><code>Typescript</code></summary>

  ```ts
  const tomorrow = new Date(Date.now() + 1000 * 60 * 60 * 24);
  // schedule a task to run tomorrow
  const scheduled = simple.schedule(tomorrow, {
    Message: "Hello, World!",
  });

  // schedule a task to run every day at midnight
  const cron = simple.cron("every-day", "0 0 * * *", {
    Message: "Hello, World!",
  });
  ```

  </details>

- <details>

    <summary><code>Go</code></summary>

  ```go
  const tomorrow = time.Now().Add(24 * time.Hour);

  // schedule a task to run tomorrow
  simple.Schedule(ctx, tomorrow, ScheduleInput{
    Message: "Hello, World!",
  })

  // schedule a task to run every day at midnight
  simple.Cron(ctx, "every-day", "0 0 * * *", CronInput{
    Message: "Hello, World!",
  })
  ```

  </details>

</details>
<details><summary><strong>🚏 Task routing</strong></summary>

####

While the default Hatchet behavior is to implement a FIFO queue, it also supports additional scheduling mechanisms to route your tasks to the ideal worker.

- **Sticky assignment** — allows spawned tasks to prefer or require execution on the same worker. [Read more ➶](https://docs.hatchet.run/home/sticky-assignment)

- **Worker affinity** — ranks workers to discover which is best suited to handle a given task. [Read more ➶](https://docs.hatchet.run/home/worker-affinity)

- <details>

    <summary><code>Python</code></summary>

  ```python
  # create a workflow which prefers to run on the same worker, but can be
  # scheduled on any worker if the original worker is busy
  hatchet.workflow(
    name="StickyWorkflow",
    sticky=StickyStrategy.SOFT,
  )

  # create a workflow which must run on the same worker
  hatchet.workflow(
    name="StickyWorkflow",
    sticky=StickyStrategy.HARD,
  )
  ```

  </details>

- <details>

    <summary><code>Typescript</code></summary>

  ```ts
  // create a workflow which prefers to run on the same worker, but can be
  // scheduled on any worker if the original worker is busy
  hatchet.workflow({
    name: "StickyWorkflow",
    sticky: StickyStrategy.SOFT,
  });

  // create a workflow which must run on the same worker
  hatchet.workflow({
    name: "StickyWorkflow",
    sticky: StickyStrategy.HARD,
  });
  ```

  </details>

- <details>

    <summary><code>Go</code></summary>

  ```go
  // create a workflow which prefers to run on the same worker, but can be
  // scheduled on any worker if the original worker is busy
  factory.NewWorkflow[StickyInput, StickyOutput](
    create.WorkflowCreateOpts[StickyInput]{
      Name: "sticky-dag",
      StickyStrategy: types.StickyStrategy_SOFT,
    },
    hatchet,
  );

  // create a workflow which must run on the same worker
  factory.NewWorkflow[StickyInput, StickyOutput](
    create.WorkflowCreateOpts[StickyInput]{
      Name: "sticky-dag",
      StickyStrategy: types.StickyStrategy_HARD,
    },
    hatchet,
  );
  ```

  </details>

</details>
<details><summary><strong>⚡️ Event triggers and listeners</strong></summary>

####

Hatchet supports event-based architectures where tasks and workflows can pause execution while waiting for a specific external event. It supports the following features:

- **Event listening** — tasks can be paused until a specific event is triggered. [Read more ➶](https://docs.hatchet.run/home/durable-execution)
- **Event triggering** — events can trigger new workflows or steps in a workflow. [Read more ➶](https://docs.hatchet.run/home/run-on-event)

- <details>

    <summary><code>Python</code></summary>

  ```python
  # Create a task which waits for an external user event or sleeps for 10 seconds
  @dag_with_conditions.task(
    parents=[first_task],
    wait_for=[
      or_(
        SleepCondition(timedelta(seconds=10)),
        UserEventCondition(event_key="user:event"),
      )
    ]
  )
  def second_task(input: EmptyModel, ctx: Context) -> dict[str, str]:
      return {"completed": "true"}
  ```

  </details>

- <details>

    <summary><code>Typescript</code></summary>

  ```ts
  // Create a task which waits for an external user event or sleeps for 10 seconds
  dagWithConditions.task({
    name: "secondTask",
    parents: [firstTask],
    waitFor: Or({ eventKey: "user:event" }, { sleepFor: "10s" }),
    fn: async (_, ctx) => {
      return {
        Completed: true,
      };
    },
  });
  ```

  </details>

- <details>

    <summary><code>Go</code></summary>

  ```go
  // Create a task which waits for an external user event or sleeps for 10 seconds
  simple.Task(
    conditionOpts{
      Name: "Step2",
      Parents: []create.NamedTask{
        step1,
      },
      WaitFor: condition.Conditions(
        condition.UserEventCondition("user:event", "'true'"),
        condition.SleepCondition(10 * time.Second),
      ),
    }, func(ctx worker.HatchetContext, input DagWithConditionsInput) (interface{}, error) {
      // ...
    },
  );
  ```

  </details>

</details>
<details><summary><strong>🖥️ Real-time Web UI</strong></summary>

####

Hatchet comes bundled with a number of features to help you monitor your tasks, workflows, and queues.

**Real-time dashboards and metrics**

Monitor your tasks, workflows, and queues with live updates to quickly detect issues. Alerting is built in so you can respond to problems as soon as they occur.

https://github.com/user-attachments/assets/b1797540-c9da-4057-b50f-4780f52a2cb9

**Logging**

Hatchet supports logging from your tasks, allowing you to easily correlate task failures with logs in your system. No more digging through your logging service to figure out why your tasks failed.

https://github.com/user-attachments/assets/427c15cd-8842-4b54-ab2e-3b1cabc01c7b

**Alerting**

Hatchet supports Slack and email-based alerting for when your tasks fail. Alerts are real-time with adjustable alerting windows.

</details>

### Quick Start

Hatchet is available as a cloud version or self-hosted. See the following docs to get up and running quickly:

- [Hatchet Cloud Quickstart](https://docs.hatchet.run/home/hatchet-cloud-quickstart)
- [Hatchet Self-Hosted](https://docs.hatchet.run/self-hosting)

### Documentation

The most up-to-date documentation can be found at https://docs.hatchet.run.

### Community & Support

- [Discord](https://discord.gg/ZMeUafwH89) - best for getting in touch with the maintainers and hanging with the community
- [Github Issues](https://github.com/hatchet-dev/hatchet/issues) - used for filing bug reports
- [Github Discussions](https://github.com/hatchet-dev/hatchet/discussions) - used for starting in-depth technical discussions that are suited for asynchronous communication
- [Email](mailto:contact@hatchet.run) - best for getting Hatchet Cloud support and for help with billing, data deletion, etc.

### Hatchet vs...

<details>
<summary>Hatchet vs Temporal</summary>

####

Hatchet is designed to be a general-purpose task orchestration platform -- it can be used as a queue, a DAG-based orchestrator, a durable execution engine, or all three. As a result, Hatchet covers a wider array of use-cases, like multiple queueing strategies, rate limiting, DAG features, conditional triggering, streaming features, and much more.

Temporal is narrowly focused on durable execution, and supports a wider range of database backends and result stores, like Apache Cassandra, MySQL, PostgreSQL, and SQLite.

**When to use Hatchet:** when you'd like to get more control over the underlying queue logic, run DAG-based workflows, or want to simplify self-hosting by only running the Hatchet engine and Postgres.

**When to use Temporal:** when you'd like to use a non-Postgres result store, or your only workload is best suited for durable execution.

</details>

<details>

<summary>Hatchet vs Task Queues (BullMQ, Celery)</summary>

####

Hatchet is a durable task queue, meaning it persists the history of all executions (up to a retention period), which allows for easy monitoring + debugging and powers a bunch of the durability features above. This isn’t the standard behavior of Celery and BullMQ (and you need to rely on third-party UI tools which are extremely limited in functionality, like Celery Flower).

**When to use Hatchet:** when you'd like results to be persisted and observable in a UI

**When to use task queue library like BullMQ/Celery:** when you need very high throughput (>10k/s) without retention, or when you'd like to use a single library (instead of a standalone service like Hatchet) to interact with your queue.

</details>

<details>

<summary>Hatchet vs DAG-based platforms (Airflow, Prefect, Dagster)</summary>

####

These tools are usually built with data engineers in mind, and aren’t designed to run as part of a high-volume application. They’re usually higher latency and higher cost, with their primary selling point being integrations with common datastores and connectors.

**When to use Hatchet:** when you'd like to use a DAG-based framework, write your own integrations and functions, and require higher throughput (>100/s)

**When to use other DAG-based platforms:** when you'd like to use other data stores and connectors that work out of the box

</details>

<details>
<summary>Hatchet vs AI Frameworks</summary>

####

Most AI frameworks are built to run in-memory, with horizontal scaling and durability as an afterthought. While you can use an AI framework in conjunction with Hatchet, most of our users discard their AI framework and use Hatchet’s primitives to build their applications.

**When to use Hatchet:** when you'd like full control over your underlying functions and LLM calls, or you require high availability and durability for your functions.

**When to use an AI framework:** when you'd like to get started quickly with simple abstractions.

</details>

### Issues

Please submit any bugs that you encounter via Github issues.

### I'd Like to Contribute

Please let us know what you're interesting in working on in the #contributing channel on [Discord](https://discord.gg/ZMeUafwH89). This will help us shape the direction of the project and will make collaboration much easier!

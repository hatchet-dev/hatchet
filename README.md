<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://framerusercontent.com/images/KBMnpSO12CyE6UANhf4mhrg6na0.png?scale-down-to=200">
  <source media="(prefers-color-scheme: light)" srcset="https://framerusercontent.com/images/KBMnpSO12CyE6UANhf4mhrg6na0.png?scale-down-to=200">
  <a href ="https://hatchet.run">
	  <img alt="Hatchet Logo" src="https://framerusercontent.com/images/KBMnpSO12CyE6UANhf4mhrg6na0.png?scale-down-to=200">
  </a>
</picture>

### Run Background Tasks at Scale

[![Docs](https://img.shields.io/badge/docs-docs.hatchet.run-3F16E4)](https://docs.hatchet.run) [![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT) [![Go Reference](https://pkg.go.dev/badge/github.com/hatchet-dev/hatchet.svg)](https://pkg.go.dev/github.com/hatchet-dev/hatchet) [![NPM Downloads](https://img.shields.io/npm/dm/%40hatchet-dev%2Ftypescript-sdk)](https://www.npmjs.com/package/@hatchet-dev/typescript-sdk)

[![Discord](https://img.shields.io/discord/1088927970518909068?style=social&logo=discord)](https://discord.gg/ZMeUafwH89)
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

Hatchet is a platform for running background tasks, built on top of Postgres. Instead of managing your own task queue or pub/sub system, you can use Hatchet to distribute your functions between a set of workers with minimal configuration or infrastructure -- the only required dependency is Postgres.

It includes the following features:

<details><summary><strong>Queues</strong></summary>

Hatchet is a durable task queue, which means that we ingest your tasks and send them to your workers at a rate that your workers can handle. Hatchet will track the progress of your task and ensure that the work gets completed (or you get alerted), even if your application crashes.

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
  @hatchet.task(name="SimpleWorkflow")
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
<details><summary><strong>Task Orchestration</strong></summary>

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
      first_result = ctx.get_parent_output(task_1)
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
<details><summary><strong>Flow Control</strong></summary>

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

### Example Use Cases

- **AI Agents:** define your agentic workflows as code and leverage Hatchet to retry failures and parallelize agent actions.
- **Fairness for Generative AI:** Don't let busy users overwhelm your system. Hatchet lets you distribute requests to your workers fairly with configurable policies.
- **Batch Processing for Document Indexing:** Hatchet can handle large-scale batch processing of documents, images, and other data and resume mid-job on failure.
- **Workflow Orchestration for Multi-Modal Systems:** Hatchet can handle orchestrating multi-modal inputs and outputs, with full DAG-style execution.
- **Correctness for Event-Based Processing:** Respond to external events or internal events within your system and replay events automatically.

### SDKs

If you encounter any issues while using the SDKs, please [submit an issue](https://github.com/hatchet-dev/hatchet/issues):

### Hatchet vs...

#### Hatchet vs Temporal

Hatchet is designed to be a general-purpose task orchestration platform -- it can be used as a queue, a DAG-based orchestrator, a durable execution engine, or all three. As a result, Hatchet covers a wider array of use-cases, like multiple queueing strategies, rate limiting, DAG features like conditional triggering, streaming features, and much more.

Temporal is narrowly focused on the durable execution pattern, but supports a wider range of database backends and result stores, like Apache Cassandra, MySQL, PostgreSQL, and SQLite.

**When to use Hatchet:** when you'd like to get more control over the underlying queue, run DAG-based workflows, or want to simplify self-hosting by only running the Hatchet engine and Postgres.

**When to use Temporal:** when you'd like to use a non-Postgres result store, or your only workload is best suited for durable execution.

#### Hatchet vs Task Queues (BullMQ, Celery)

Hatchet is a durable task queue, meaning it persists the history of all executions (up to a retention period), which allows for easy monitoring + debugging and powers a bunch of the durability features above. This isn’t the standard behavior of Celery and BullMQ (and you need to rely on third-party UI tools which are extremely limited in functionality, like Celery Flower).

**When to use Hatchet:** when you'd like to results to be persisted and observable in a UI

**When to use task queue library like BullMQ/Celery:** when you'd like to use a single library (instead of a standalone service) to interact with your queue

#### Hatchet vs DAG-based platforms (Airflow, Prefect, Dagster)

These tools are usually built with data engineers in mind, and aren’t meant to run from a high-volume application. They’re usually lower latency and higher cost, with their primary selling point being integrations with common datastores and connectors.

#### Hatchet vs AI Frameworks

Most AI frameworks are built to run in-memory, with horizontal scaling and durability as an afterthought. While you can use an AI framework in conjunction with Hatchet, most of our users discard their AI framework and use Hatchet’s primitives to build their applications.

### Issues

Please submit any bugs that you encounter via Github issues. However, please reach out on [Discord](https://discord.gg/ZMeUafwH89) before submitting a feature request - as the project is very early, we'd like to build a solid foundation before adding more complex features.

### I'd Like to Contribute

Please let us know what you're interesting in working on in the #contributing channel on [Discord](https://discord.gg/ZMeUafwH89). This will help us shape the direction of the project and will make collaboration much easier!

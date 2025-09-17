# Context

The Hatchet Context class provides helper methods and useful data to tasks at runtime. It is passed as the second argument to all tasks and durable tasks.

There are two types of context classes you'll encounter:

* `Context` - The standard context for regular tasks with methods for logging, task output retrieval, cancellation, and more
* `DurableContext` - An extended context for durable tasks that includes additional methods for durable execution like `aio_wait_for` and `aio_sleep_for`


## Context

::: context.context.Context
    options:
      inherited_members: false
      members:
        - was_skipped
        - task_output
        - was_triggered_by_event
        - workflow_input
        - lifespan
        - workflow_run_id
        - cancel
        - aio_cancel
        - done
        - log
        - release_slot
        - put_stream
        - refresh_timeout
        - retry_count
        - attempt_number
        - additional_metadata
        - parent_workflow_run_id
        - priority
        - workflow_id
        - workflow_version_id
        - task_run_errors
        - fetch_task_run_error

## DurableContext

::: context.context.DurableContext
    options:
      inherited_members: true
      members:
        - aio_wait_for
        - aio_sleep_for

# Runnables

`Runnables` in the Hatchet SDK are things that can be run, namely tasks and workflows. The two main types of runnables you'll encounter are:

* `Workflow`, which lets you define tasks and call all of the run, schedule, etc. methods
* `Standalone`, which is a single task that's returned by `hatchet.task` and can be run, scheduled, etc.

## Workflow

::: runnables.workflow.Workflow
    options:
      inherited_members: true
      members:
        - task
        - batch_task
        - durable_task
        - on_failure_task
        - on_success_task
        - run
        - aio_run
        - run_no_wait
        - aio_run_no_wait
        - run_many
        - aio_run_many
        - run_many_no_wait
        - aio_run_many_no_wait
        - schedule
        - aio_schedule
        - create_cron
        - aio_create_cron
        - create_bulk_run_item
        - name
        - tasks
        - id
        - list_runs
        - aio_list_runs
        - create_filter
        - aio_create_filter

## Task

::: runnables.task.Task
    options:
      inherited_members: true
      members:
        - mock_run
        - aio_mock_run

## Standalone

::: runnables.workflow.Standalone
    options:
      inherited_members: true
      members:
        - run
        - aio_run
        - run_no_wait
        - aio_run_no_wait
        - run_many
        - aio_run_many
        - run_many_no_wait
        - aio_run_many_no_wait
        - schedule
        - aio_schedule
        - create_cron
        - aio_create_cron
        - create_bulk_run_item
        - list_runs
        - aio_list_runs
        - create_filter
        - aio_create_filter
        - delete
        - aio_delete
        - get_run_ref
        - get_result
        - aio_get_result
        - mock_run
        - aio_mock_run

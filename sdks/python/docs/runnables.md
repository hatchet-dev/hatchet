# Runnables

`Runnables` in the Hatchet SDK are things that can be run, namely tasks and workflows. The two main types of runnables you'll encounter are:

* `Workflow`, which lets you define tasks and call all of the run, schedule, etc. methods
* `Standalone`, which is a single task that's returned by `hatchet.task` and can be run, scheduled, etc.

## Workflow

::: runnables.workflow.Workflow
    options:
      members:
        - task
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
        - is_durable

## Standalone

::: runnables.standalone.Standalone
    options:
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
        - is_durable

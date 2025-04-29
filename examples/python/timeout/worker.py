import time
from datetime import timedelta

from hatchet_sdk import Context, EmptyModel, Hatchet, TaskDefaults

hatchet = Hatchet(debug=True)

# > ScheduleTimeout
timeout_wf = hatchet.workflow(
    name="TimeoutWorkflow",
    task_defaults=TaskDefaults(execution_timeout=timedelta(minutes=2)),
)


# > ExecutionTimeout
# 👀 Specify an execution timeout on a task
@timeout_wf.task(
    execution_timeout=timedelta(seconds=4), schedule_timeout=timedelta(minutes=10)
)
def timeout_task(input: EmptyModel, ctx: Context) -> dict[str, str]:
    time.sleep(5)
    return {"status": "success"}



refresh_timeout_wf = hatchet.workflow(name="RefreshTimeoutWorkflow")


# > RefreshTimeout
@refresh_timeout_wf.task(execution_timeout=timedelta(seconds=4))
def refresh_task(input: EmptyModel, ctx: Context) -> dict[str, str]:

    ctx.refresh_timeout(timedelta(seconds=10))
    time.sleep(5)

    return {"status": "success"}




def main() -> None:
    worker = hatchet.worker(
        "timeout-worker", slots=4, workflows=[timeout_wf, refresh_timeout_wf]
    )

    worker.start()


if __name__ == "__main__":
    main()

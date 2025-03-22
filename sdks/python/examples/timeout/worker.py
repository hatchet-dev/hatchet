import time
from datetime import timedelta

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

timeout_wf = hatchet.workflow(name="TimeoutWorkflow")


@timeout_wf.task(execution_timeout=timedelta(seconds=4))
def timeout_task(input: EmptyModel, ctx: Context) -> dict[str, str]:
    time.sleep(5)
    return {"status": "success"}


refresh_timeout_wf = hatchet.workflow(name="RefreshTimeoutWorkflow")


@refresh_timeout_wf.task(execution_timeout=timedelta(seconds=4))
def refresh_task(input: EmptyModel, ctx: Context) -> dict[str, str]:

    ctx.refresh_timeout("10s")
    time.sleep(5)

    return {"status": "success"}


def main() -> None:
    worker = hatchet.worker(
        "timeout-worker", slots=4, workflows=[timeout_wf, refresh_timeout_wf]
    )

    worker.start()


if __name__ == "__main__":
    main()

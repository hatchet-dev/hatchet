from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

simple_workflow = hatchet.workflow(name="SimpleRetryWorkflow")
backoff_workflow = hatchet.workflow(name="BackoffWorkflow")


# > Simple Step Retries
@simple_workflow.task(retries=3)
def always_fail(input: EmptyModel, ctx: Context) -> dict[str, str]:
    raise Exception("simple task failed")




# > Retries with Count
@simple_workflow.task(retries=3)
def fail_twice(input: EmptyModel, ctx: Context) -> dict[str, str]:
    if ctx.retry_count < 2:
        raise Exception("simple task failed")

    return {"status": "success"}




# > Retries with Backoff
@backoff_workflow.task(
    retries=10,
    # 👀 Maximum number of seconds to wait between retries
    backoff_max_seconds=10,
    # 👀 Factor to increase the wait time between retries.
    # This sequence will be 2s, 4s, 8s, 10s, 10s, 10s... due to the maxSeconds limit
    backoff_factor=2.0,
)
def backoff_task(input: EmptyModel, ctx: Context) -> dict[str, str]:
    if ctx.retry_count < 3:
        raise Exception("backoff task failed")

    return {"status": "success"}




def main() -> None:
    worker = hatchet.worker("backoff-worker", slots=4, workflows=[backoff_workflow])
    worker.start()


if __name__ == "__main__":
    main()

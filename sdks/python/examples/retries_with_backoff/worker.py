from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)
backoff_workflow = hatchet.workflow(name="BackoffWorkflow", on_events=["backoff:event"])


# ❓ Backoff
# 👀 Backoff configuration
@backoff_workflow.task(
    retries=10,
    # 👀 Maximum number of seconds to wait between retries
    backoff_max_seconds=60,
    # 👀 Factor to increase the wait time between retries.
    # This sequence will be 2s, 4s, 8s, 16s, 32s, 60s... due to the maxSeconds limit
    backoff_factor=2.0,
)
def step1(input: EmptyModel, context: Context) -> dict[str, str]:
    if context.retry_count < 3:
        raise Exception("step1 failed")

    return {"status": "success"}


# ‼️


def main() -> None:
    worker = hatchet.worker("backoff-worker", slots=4, workflows=[backoff_workflow])
    worker.start()


if __name__ == "__main__":
    main()

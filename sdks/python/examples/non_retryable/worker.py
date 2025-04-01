from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.runnables.task import NonRetryableException

hatchet = Hatchet(debug=True)

non_retryable_workflow = hatchet.workflow(name="NonRetryableWorkflow")


@non_retryable_workflow.task(
    skip_retry_on_exceptions=[
        NonRetryableException(exception=ValueError, match="foobar")
    ],
    retries=1,
)
def should_not_retry(input: EmptyModel, ctx: Context) -> None:
    raise ValueError("foobar")


@non_retryable_workflow.task(
    skip_retry_on_exceptions=[
        NonRetryableException(exception=ValueError, match="foobar")
    ],
    retries=1,
)
def should_be_retried(input: EmptyModel, ctx: Context) -> None:
    raise TypeError("foobar")


def main() -> None:
    worker = hatchet.worker("non-retry-worker", workflows=[non_retryable_workflow])

    worker.start()


if __name__ == "__main__":
    main()

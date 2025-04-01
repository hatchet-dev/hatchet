from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.runnables.task import NonRetryableException

hatchet = Hatchet(debug=True)

non_retryable_workflow = hatchet.workflow(name="NonRetryableWorkflow")


@non_retryable_workflow.task(
    skip_retry_on_exceptions=[
        NonRetryableException(exception=ValueError, match="should not retry")
    ],
    retries=1,
)
def should_not_retry(input: EmptyModel, ctx: Context) -> None:
    raise ValueError(
        "This error should not retry because the match clause matches the exception."
    )


@non_retryable_workflow.task(
    skip_retry_on_exceptions=[
        NonRetryableException(exception=ValueError, match="should retry")
    ],
    retries=1,
)
def should_retry_wrong_exception_type(input: EmptyModel, ctx: Context) -> None:
    raise TypeError(
        "This error should retry because the exception type does not match."
    )


@non_retryable_workflow.task(
    skip_retry_on_exceptions=[
        NonRetryableException(exception=ValueError, match="bazqux")
    ],
    retries=1,
)
def should_retry_wrong_match(input: EmptyModel, ctx: Context) -> None:
    raise ValueError(
        "This error should not retry because the match clause does not match."
    )


@non_retryable_workflow.task(
    skip_retry_on_exceptions=[NonRetryableException(exception=ValueError)],
    retries=1,
)
def should_not_retry_no_match(input: EmptyModel, ctx: Context) -> None:
    raise ValueError(
        "This error should not retry because there was no match clause provided."
    )


@non_retryable_workflow.task(
    skip_retry_on_exceptions=[
        NonRetryableException(exception=ValueError, match="foobar")
    ],
    retries=1,
)
def should_retry_no_match(input: EmptyModel, ctx: Context) -> None:
    raise ValueError("This error should retry because the match clause does not match.")


def main() -> None:
    worker = hatchet.worker("non-retry-worker", workflows=[non_retryable_workflow])

    worker.start()


if __name__ == "__main__":
    main()

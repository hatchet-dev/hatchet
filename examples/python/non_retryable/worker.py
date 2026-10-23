from hatchet_sdk import Context, Hatchet
from hatchet_sdk.exceptions import NonRetryableError

hatchet = Hatchet()

non_retryable_workflow = hatchet.workflow(name="NonRetryableWorkflow")


# > Non-retryable task
@non_retryable_workflow.task(retries=1)
def should_not_retry(input: None, ctx: Context) -> None:
    raise NonRetryableError("This task should not retry")




@non_retryable_workflow.task(retries=1)
def should_retry_wrong_exception_type(input: None, ctx: Context) -> None:
    raise TypeError("This task should retry because it's not a NonRetryableError")


@non_retryable_workflow.task(retries=1)
def should_not_retry_successful_task(input: None, ctx: Context) -> None:
    pass


def main() -> None:
    worker = hatchet.worker("non-retry-worker", workflows=[non_retryable_workflow])

    worker.start()


if __name__ == "__main__":
    main()

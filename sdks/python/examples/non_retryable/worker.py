from hatchet_sdk import Context, EmptyModel, Hatchet, NonRetryableException

hatchet = Hatchet(debug=True)

non_retryable_workflow = hatchet.workflow(name="NonRetryableWorkflow")


# ❓ Non-retryable task
@non_retryable_workflow.task(retries=1)
def should_not_retry(input: EmptyModel, ctx: Context) -> None:
    raise NonRetryableException("This task should not retry")
# !!

@non_retryable_workflow.task(retries=1)
def should_retry_wrong_exception_type(input: EmptyModel, ctx: Context) -> None:
    raise TypeError("This task should retry because it's not a NonRetryableException")


@non_retryable_workflow.task(retries=1)
def should_not_retry_successful_task(input: EmptyModel, ctx: Context) -> None:
    pass


def main() -> None:
    worker = hatchet.worker("non-retry-worker", workflows=[non_retryable_workflow])

    worker.start()


if __name__ == "__main__":
    main()

import random
from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet, RetryAfterException

hatchet = Hatchet()


class UpstreamInput(BaseModel):
    failing_attempts: int = 2


# > Retry after an upstream-provided delay
@hatchet.task(name="RetryAfterUpstreamDelay", input_validator=UpstreamInput, retries=5)
def retry_after_upstream_delay(input: UpstreamInput, ctx: Context) -> dict[str, int]:
    if ctx.retry_count < input.failing_attempts:
        raise RetryAfterException("upstream rate limited", after=timedelta(seconds=2))

    return {"attempt": ctx.retry_count}


# !!


# > Exponential backoff with full jitter
@hatchet.task(
    name="RetryAfterExponentialBackoff", input_validator=UpstreamInput, retries=3
)
def retry_after_exponential_backoff(
    input: UpstreamInput, ctx: Context
) -> dict[str, int]:
    if ctx.retry_count < input.failing_attempts:
        raise RetryAfterException(
            "upstream unavailable",
            after=random.uniform(0, min(2**ctx.retry_count, 60)),
        )

    return {"attempt": ctx.retry_count}


# !!


@hatchet.task(
    name="RetryAfterOverridesBackoff",
    input_validator=UpstreamInput,
    retries=2,
    backoff_factor=10,
    backoff_max_seconds=60,
)
def retry_after_overrides_backoff(input: UpstreamInput, ctx: Context) -> dict[str, int]:
    if ctx.retry_count < input.failing_attempts:
        raise RetryAfterException("upstream unavailable", after=timedelta(seconds=1))

    return {"attempt": ctx.retry_count}


def main() -> None:
    worker = hatchet.worker(
        "retry-after-worker",
        workflows=[
            retry_after_upstream_delay,
            retry_after_exponential_backoff,
            retry_after_overrides_backoff,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()

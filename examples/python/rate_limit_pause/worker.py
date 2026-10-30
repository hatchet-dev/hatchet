from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
)
from hatchet_sdk.rate_limit import RateLimit, RateLimitDuration

hatchet = Hatchet()


class RateLimitPauseInput(BaseModel):
    group: str
    key: str


@hatchet.task(
    input_validator=RateLimitPauseInput,
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
    rate_limits=[
        RateLimit(
            dynamic_key="input.key",
            units=1,
            limit=10,
            duration=RateLimitDuration.HOUR,
        )
    ],
)
async def rate_limit_pause_task(input: RateLimitPauseInput, ctx: Context) -> None:
    return None

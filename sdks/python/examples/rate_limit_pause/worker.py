from pydantic import BaseModel

from hatchet_sdk import ConcurrencyStrategy, Context, Hatchet, RateLimit

hatchet = Hatchet()


class RateLimitPauseInput(BaseModel):
    group: str
    key: str


@hatchet.task(
    input_validator=RateLimitPauseInput,
    concurrency=ConcurrencyStrategy(
        expression="input.group",
        max_runs=1,
        strategy="GROUP_ROUND_ROBIN",
    ),
    rate_limits=[
        RateLimit(
            dynamic_key="input.key",
            units=1,
            limit=10,
            duration="HOUR",
        )
    ],
)
async def rate_limit_pause_task(input: RateLimitPauseInput, ctx: Context) -> None:
    return None

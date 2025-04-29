from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.rate_limit import RateLimit, RateLimitDuration

hatchet = Hatchet(debug=True)


# > Workflow
class RateLimitInput(BaseModel):
    user_id: str


rate_limit_workflow = hatchet.workflow(
    name="RateLimitWorkflow", input_validator=RateLimitInput
)




# > Static
RATE_LIMIT_KEY = "test-limit"


@rate_limit_workflow.task(rate_limits=[RateLimit(static_key=RATE_LIMIT_KEY, units=1)])
def step_1(input: RateLimitInput, ctx: Context) -> None:
    print("executed step_1")




# > Dynamic


@rate_limit_workflow.task(
    rate_limits=[
        RateLimit(
            dynamic_key="input.user_id",
            units=1,
            limit=10,
            duration=RateLimitDuration.MINUTE,
        )
    ]
)
def step_2(input: RateLimitInput, ctx: Context) -> None:
    print("executed step_2")





def main() -> None:
    hatchet.rate_limits.put(RATE_LIMIT_KEY, 2, RateLimitDuration.SECOND)

    worker = hatchet.worker(
        "rate-limit-worker", slots=10, workflows=[rate_limit_workflow]
    )

    worker.start()


if __name__ == "__main__":
    main()

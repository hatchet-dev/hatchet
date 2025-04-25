from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.rate_limit import RateLimit

hatchet = Hatchet(debug=True)

class DynamicRateLimitInput(BaseModel):
    group: str
    units: int
    limit: int

dynamic_rate_limit_workflow = hatchet.workflow(
    name="DynamicRateLimitWorkflow", input_validator=DynamicRateLimitInput
)

@dynamic_rate_limit_workflow.task(
    rate_limits=[
        RateLimit(
            dynamic_key='"LIMIT:"+input.group',
            units="input.units",
            limit="input.limit",
        )
    ]
)
def step1(input: DynamicRateLimitInput, ctx: Context) -> None:
    print("executed step1")

def main() -> None:
    worker = hatchet.worker(
        "rate-limit-worker", slots=10, workflows=[dynamic_rate_limit_workflow]
    )
    worker.start()

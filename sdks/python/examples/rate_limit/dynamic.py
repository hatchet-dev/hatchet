from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.rate_limit import RateLimit

hatchet = Hatchet(debug=True)

wf = hatchet.workflow(name="RateLimitWorkflow", on_events=["rate_limit:create"])


@wf.task(
    rate_limits=[
        RateLimit(
            dynamic_key='"LIMIT:"+input.group',
            units="input.units",
            limit="input.limit",
        )
    ]
)
def step1(input: EmptyModel, context: Context) -> None:
    print("executed step1")


def main() -> None:
    worker = hatchet.worker("rate-limit-worker", slots=10, workflows=[wf])
    worker.start()

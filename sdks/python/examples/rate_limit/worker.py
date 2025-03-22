from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.rate_limit import RateLimit, RateLimitDuration

hatchet = Hatchet(debug=True)
rate_limit_workflow = hatchet.workflow(name="RateLimitWorkflow")

RATE_LIMIT_KEY = "test-limit"


@rate_limit_workflow.task(rate_limits=[RateLimit(static_key=RATE_LIMIT_KEY, units=1)])
def step1(input: EmptyModel, ctx: Context) -> None:
    print("executed step1")


def main() -> None:
    # hatchet.admin.put_rate_limit(RATE_LIMIT_KEY, 2, RateLimitDuration.SECOND)

    worker = hatchet.worker(
        "rate-limit-worker", slots=10, workflows=[rate_limit_workflow]
    )

    worker.start()


if __name__ == "__main__":
    main()

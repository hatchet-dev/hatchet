from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.rate_limit import RateLimit, RateLimitDuration

hatchet = Hatchet(debug=True)
wf = hatchet.workflow(name="RateLimitWorkflow", on_events=["rate_limit:create"])


@wf.task(rate_limits=[RateLimit(static_key="test-limit", units=1)])
def step1(input: EmptyModel, context: Context) -> None:
    print("executed step1")


def main() -> None:
    hatchet.admin.put_rate_limit("test-limit", 2, RateLimitDuration.SECOND)

    worker = hatchet.worker("rate-limit-worker", slots=10, workflows=[wf])

    worker.start()


if __name__ == "__main__":
    main()

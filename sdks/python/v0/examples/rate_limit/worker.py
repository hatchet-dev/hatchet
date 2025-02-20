from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.rate_limit import RateLimit, RateLimitDuration

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["rate_limit:create"])
class RateLimitWorkflow:

    @hatchet.step(rate_limits=[RateLimit(key="test-limit", units=1)])
    def step1(self, context: Context) -> None:
        print("executed step1")


def main() -> None:
    hatchet.admin.put_rate_limit("test-limit", 2, RateLimitDuration.SECOND)

    worker = hatchet.worker("rate-limit-worker", max_runs=10)
    worker.register_workflow(RateLimitWorkflow())

    worker.start()


if __name__ == "__main__":
    main()

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.rate_limit import RateLimit, RateLimitDuration

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["rate_limit:create"])
class RateLimitWorkflow:
    def __init__(self):
        self.my_value = "test"

    @hatchet.step(rate_limits=[RateLimit(key="test-limit", units=1)])
    def step1(self, context: Context):
        print("executed step1")
        pass


hatchet.client.admin.put_rate_limit("test-limit", 2, RateLimitDuration.MINUTE)

worker = hatchet.worker("test-worker", max_runs=4)
worker.register_workflow(RateLimitWorkflow())

worker.start()

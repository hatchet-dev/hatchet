import hashlib
import time

from hatchet_sdk import BaseWorkflow, Context, Hatchet

hatchet = Hatchet(debug=True)

# WARNING: this is an example of what NOT to do
# This workflow is intentionally blocking the main thread
# and will block the worker from processing other workflows
#
# You do not want to run long sync functions in an async def function

wf = hatchet.declare_workflow(on_events=["user:create"])


class Blocked(BaseWorkflow):
    config = wf.config

    @hatchet.step(timeout="11s", retries=3)
    async def step1(self, context: Context) -> dict[str, str | int | float]:
        print("Executing step1")

        # CPU-bound task: Calculate a large number of SHA-256 hashes
        start_time = time.time()
        iterations = 10_000_000
        for i in range(iterations):
            hashlib.sha256(f"data{i}".encode()).hexdigest()

        end_time = time.time()
        execution_time = end_time - start_time

        print(
            f"Completed {iterations} hash calculations in {execution_time:.2f} seconds"
        )

        return {
            "step1": "step1",
            "iterations": iterations,
            "execution_time": execution_time,
        }


def main() -> None:
    worker = hatchet.worker("blocked-worker", max_runs=3)
    worker.register_workflow(Blocked())
    worker.start()


if __name__ == "__main__":
    main()

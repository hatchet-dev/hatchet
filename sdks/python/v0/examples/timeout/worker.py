import time

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["timeout:create"])
class TimeoutWorkflow:

    @hatchet.step(timeout="4s")
    def step1(self, context: Context) -> dict[str, str]:
        time.sleep(5)
        return {"status": "success"}


@hatchet.workflow(on_events=["refresh:create"])
class RefreshTimeoutWorkflow:

    @hatchet.step(timeout="4s")
    def step1(self, context: Context) -> dict[str, str]:

        context.refresh_timeout("10s")
        time.sleep(5)

        return {"status": "success"}


def main() -> None:
    worker = hatchet.worker("timeout-worker", max_runs=4)
    worker.register_workflow(TimeoutWorkflow())
    worker.register_workflow(RefreshTimeoutWorkflow())

    worker.start()


if __name__ == "__main__":
    main()

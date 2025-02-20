import time

from hatchet_sdk import BaseWorkflow, Context, Hatchet

hatchet = Hatchet(debug=True)

timeout_wf = hatchet.declare_workflow(on_events=["timeout:create"])


class TimeoutWorkflow(BaseWorkflow):
    config = timeout_wf.config

    @hatchet.step(timeout="4s")
    def step1(self, context: Context) -> dict[str, str]:
        time.sleep(5)
        return {"status": "success"}


refresh_timeout_wf = hatchet.declare_workflow(on_events=["refresh:create"])


class RefreshTimeoutWorkflow(BaseWorkflow):
    config = refresh_timeout_wf.config

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

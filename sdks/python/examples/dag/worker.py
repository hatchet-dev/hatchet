import random
import time
from typing import Any, cast

from hatchet_sdk import BaseWorkflow, Context, Hatchet

hatchet = Hatchet(debug=True)

wf = hatchet.declare_workflow(on_events=["dag:create"], schedule_timeout="10m")


class DagWorkflow(BaseWorkflow):
    config = wf.config

    @hatchet.step(timeout="5s")
    def step1(self, context: Context) -> dict[str, int]:
        rando = random.randint(
            1, 100
        )  # Generate a random number between 1 and 100return {
        return {
            "rando": rando,
        }

    @hatchet.step(timeout="5s")
    def step2(self, context: Context) -> dict[str, int]:
        rando = random.randint(
            1, 100
        )  # Generate a random number between 1 and 100return {
        return {
            "rando": rando,
        }

    @hatchet.step(parents=["step1", "step2"])
    def step3(self, context: Context) -> dict[str, int]:
        one = cast(dict[str, Any], context.step_output("step1"))["rando"]
        two = cast(dict[str, Any], context.step_output("step2"))["rando"]

        return {
            "sum": one + two,
        }

    @hatchet.step(parents=["step1", "step3"])
    def step4(self, context: Context) -> dict[str, str]:
        print(
            "executed step4",
            time.strftime("%H:%M:%S", time.localtime()),
            context.workflow_input,
            context.step_output("step1"),
            context.step_output("step3"),
        )
        return {
            "step4": "step4",
        }


def main() -> None:
    worker = hatchet.worker("dag-worker")
    worker.register_workflow(DagWorkflow())

    worker.start()


if __name__ == "__main__":
    main()

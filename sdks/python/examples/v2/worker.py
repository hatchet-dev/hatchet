from examples.v2.workflows import example_workflow, hatchet
from hatchet_sdk import BaseWorkflow, Context


class ExampleV2Workflow(BaseWorkflow):
    config = example_workflow.config

    @hatchet.step(timeout="11s", retries=3)
    def step1(self, context: Context) -> None:
        input = example_workflow.get_workflow_input(context)

        print(input.message)

        return None


def main() -> None:
    worker = hatchet.worker("test-worker", max_runs=1)
    worker.register_workflow(ExampleV2Workflow())
    worker.start()


if __name__ == "__main__":
    main()

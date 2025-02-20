import json

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)

# ❓ OnFailure Step
# This workflow will fail because the step will throw an error
# we define an onFailure step to handle this case


@hatchet.workflow(on_events=["user:create"])
class OnFailureWorkflow:
    @hatchet.step(timeout="1s")
    def step1(self, context: Context) -> None:
        # 👀 this step will always raise an exception
        raise Exception("step1 failed")

    # 👀 After the workflow fails, this special step will run
    @hatchet.on_failure_step()
    def on_failure(self, context: Context) -> dict[str, str]:
        # 👀 we can do things like perform cleanup logic
        # or notify a user here

        # 👀 Fetch the errors from upstream step runs from the context
        print(context.step_run_errors())

        return {"status": "success"}


# ‼️


# ❓ OnFailure With Details
# We can access the failure details in the onFailure step
# via the context method
@hatchet.workflow(on_events=["user:create"])
class OnFailureWorkflowWithDetails:
    # ... defined as above
    @hatchet.step(timeout="1s")
    def step1(self, context: Context) -> None:
        raise Exception("step1 failed")

    # 👀 After the workflow fails, this special step will run
    @hatchet.on_failure_step()
    def on_failure(self, context: Context) -> dict[str, str]:
        failures = context.fetch_run_failures()

        # 👀 we can access the failure details here
        print(json.dumps(failures, indent=2))
        if len(failures) == 1 and "step1 failed" in failures[0]["error"]:
            return {"status": "success"}

        raise Exception("unexpected failure")


# ‼️


def main() -> None:
    workflow = OnFailureWorkflow()
    workflow2 = OnFailureWorkflowWithDetails()
    worker = hatchet.worker("on-failure-worker", max_runs=4)
    worker.register_workflow(workflow)
    worker.register_workflow(workflow2)

    worker.start()


if __name__ == "__main__":
    main()

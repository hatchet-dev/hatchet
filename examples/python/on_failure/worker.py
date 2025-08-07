import json
from datetime import timedelta

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.exceptions import TaskRunError

hatchet = Hatchet(debug=False)

ERROR_TEXT = "step1 failed"

# > OnFailure Step
# This workflow will fail because the step will throw an error
# we define an onFailure step to handle this case

on_failure_wf = hatchet.workflow(name="OnFailureWorkflow")


@on_failure_wf.task(execution_timeout=timedelta(seconds=1))
def step1(input: EmptyModel, ctx: Context) -> None:
    # 👀 this step will always raise an exception
    raise Exception(ERROR_TEXT)


# 👀 After the workflow fails, this special step will run
@on_failure_wf.on_failure_task()
def on_failure(input: EmptyModel, ctx: Context) -> dict[str, str]:
    # 👀 we can do things like perform cleanup logic
    # or notify a user here

    # 👀 Fetch the errors from upstream step runs from the context
    print(ctx.task_run_errors)

    return {"status": "success"}




# > OnFailure With Details
# We can access the failure details in the onFailure step
# via the context method

on_failure_wf_with_details = hatchet.workflow(name="OnFailureWorkflowWithDetails")


# ... defined as above
@on_failure_wf_with_details.task(execution_timeout=timedelta(seconds=1))
def details_step1(input: EmptyModel, ctx: Context) -> None:
    raise Exception(ERROR_TEXT)


# 👀 After the workflow fails, this special step will run
@on_failure_wf_with_details.on_failure_task()
def details_on_failure(input: EmptyModel, ctx: Context) -> dict[str, str | None]:
    error = ctx.get_task_run_error(details_step1)

    if not error:
        return {"status": "unexpected success"}

    # 👀 we can access the failure details here
    assert isinstance(error, TaskRunError)

    if "step1 failed" in error.exc:
        return {
            "status": "success",
            "failed_run_external_id": error.task_run_external_id,
        }

    raise Exception("unexpected failure")




def main() -> None:
    worker = hatchet.worker(
        "on-failure-worker",
        slots=4,
        workflows=[on_failure_wf, on_failure_wf_with_details],
    )
    worker.start()


if __name__ == "__main__":
    main()

from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


# > Workflow Definition Cron Trigger
# Adding a cron trigger to a workflow is as simple
# as adding a `cron expression` to the `on_cron`
# prop of the workflow definition

cron_workflow = hatchet.workflow(name="CronWorkflow", on_crons=["* * * * *"])


class TaskOutput(BaseModel):
    message: str


@cron_workflow.task()
def step1(input: EmptyModel, ctx: Context) -> TaskOutput:
    return TaskOutput(
        message="Hello, world!",
    )




def main() -> None:
    worker = hatchet.worker("test-worker", slots=1, workflows=[cron_workflow])
    worker.start()


if __name__ == "__main__":
    main()

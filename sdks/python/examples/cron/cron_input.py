from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()


class CronInput(BaseModel):
    name: str


# > Cron Trigger with Input
# You can pass a `cron_input` to a workflow so that runs triggered by its
# `on_crons` schedules receive a predefined input.

cron_input_workflow = hatchet.workflow(
    name="CronInputWorkflow",
    input_validator=CronInput,
    on_crons=["* * * * *"],
    cron_input=CronInput(name="Hatchet"),
)


@cron_input_workflow.task()
def greet(input: CronInput, ctx: Context) -> dict[str, str]:
    return {"message": f"Hello, {input.name}!"}


# !!


def main() -> None:
    worker = hatchet.worker(
        "cron-input-worker", slots=1, workflows=[cron_input_workflow]
    )
    worker.start()


if __name__ == "__main__":
    main()

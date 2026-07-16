from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()


class CronInput(BaseModel):
    name: str


# > Cron Trigger with Input
# You can pass a `cron_input` to a workflow so that runs triggered by its
# `on_crons` schedules receive a predefined input.


@hatchet.task(
    name="CronInputWorkflow",
    input_validator=CronInput,
    on_crons=["* * * * *"],
    cron_input=CronInput(name="Hatchet"),
)
def cron_input_example_send_greeting(input: CronInput, ctx: Context) -> dict[str, str]:
    return {"message": f"Hello, {input.name}!"}




def main() -> None:
    worker = hatchet.worker(
        "cron-input-worker",
        workflows=[cron_input_example_send_greeting],
    )
    worker.start()


if __name__ == "__main__":
    main()

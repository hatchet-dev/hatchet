from hatchet_sdk import Context, Hatchet, IdempotencyConfig
from datetime import timedelta
from pydantic import BaseModel

hatchet = Hatchet()


class IdempotencyInput(BaseModel):
    id: str


EVENT_KEY = "idempotency:example"


@hatchet.task(
    idempotency=IdempotencyConfig(key_expression="input.id", ttl=timedelta(minutes=1)),
    input_validator=IdempotencyInput,
    on_events=[EVENT_KEY],
)
async def idempotent_task(input: IdempotencyInput, ctx: Context) -> dict[str, str]:
    return {"result": f"Hello, world from task {input.id}"}


@hatchet.task(
    idempotency=IdempotencyConfig(key_expression="input.id", ttl=timedelta(seconds=2)),
    input_validator=IdempotencyInput,
    on_events=[EVENT_KEY],
)
async def idempotent_task_short_window(
    input: IdempotencyInput, ctx: Context
) -> dict[str, str]:
    return {"result": f"Hello, world from task {input.id}"}


def main() -> None:
    worker = hatchet.worker(
        "test-worker",
        workflows=[idempotent_task],
    )
    worker.start()


if __name__ == "__main__":
    main()

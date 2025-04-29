# > Lifespan

from typing import AsyncGenerator, cast

from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


class Lifespan(BaseModel):
    foo: str
    pi: float


async def lifespan() -> AsyncGenerator[Lifespan, None]:
    yield Lifespan(foo="bar", pi=3.14)


@hatchet.task(name="LifespanWorkflow")
def lifespan_task(input: EmptyModel, ctx: Context) -> Lifespan:
    return cast(Lifespan, ctx.lifespan)


def main() -> None:
    worker = hatchet.worker(
        "test-worker", slots=1, workflows=[lifespan_task], lifespan=lifespan
    )
    worker.start()




if __name__ == "__main__":
    main()

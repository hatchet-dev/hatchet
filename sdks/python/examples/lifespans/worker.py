from typing import AsyncGenerator, cast
from uuid import UUID

from psycopg_pool import ConnectionPool
from pydantic import BaseModel, ConfigDict

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


# > Use the lifespan in a task
class TaskOutput(BaseModel):
    num_rows: int
    external_ids: list[UUID]


lifespan_workflow = hatchet.workflow(name="LifespanWorkflow")


@lifespan_workflow.task()
def sync_lifespan_task(input: EmptyModel, ctx: Context) -> TaskOutput:
    pool = cast(Lifespan, ctx.lifespan).pool

    with pool.connection() as conn:
        query = conn.execute("SELECT * FROM v1_lookup_table_olap LIMIT 5;")
        rows = query.fetchall()

        for row in rows:
            print(row)

        print("executed sync task with lifespan", ctx.lifespan)

        return TaskOutput(
            num_rows=len(rows),
            external_ids=[cast(UUID, row[0]) for row in rows],
        )


# !!


@lifespan_workflow.task()
async def async_lifespan_task(input: EmptyModel, ctx: Context) -> TaskOutput:
    pool = cast(Lifespan, ctx.lifespan).pool

    with pool.connection() as conn:
        query = conn.execute("SELECT * FROM v1_lookup_table_olap LIMIT 5;")
        rows = query.fetchall()

        for row in rows:
            print(row)

        print("executed async task with lifespan", ctx.lifespan)

        return TaskOutput(
            num_rows=len(rows),
            external_ids=[cast(UUID, row[0]) for row in rows],
        )


# > Define a lifespan
class Lifespan(BaseModel):
    model_config = ConfigDict(arbitrary_types_allowed=True)

    foo: str
    pool: ConnectionPool


async def lifespan() -> AsyncGenerator[Lifespan, None]:
    print("Running lifespan!")
    with ConnectionPool("postgres://hatchet:hatchet@localhost:5431/hatchet") as pool:
        yield Lifespan(
            foo="bar",
            pool=pool,
        )

    print("Cleaning up lifespan!")


worker = hatchet.worker(
    "test-worker", slots=1, workflows=[lifespan_workflow], lifespan=lifespan
)
# !!


def main() -> None:
    worker.start()


if __name__ == "__main__":
    main()

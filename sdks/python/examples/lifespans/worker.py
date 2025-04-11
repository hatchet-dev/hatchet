# ❓ Lifespan

from typing import Any, AsyncGenerator, cast
from uuid import UUID

from psycopg_pool import ConnectionPool
from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


class TaskOutput(BaseModel):
    num_rows: int
    external_ids: list[UUID]


async def lifespan() -> AsyncGenerator[dict[str, Any], None]:
    print("Running lifespan!")
    with ConnectionPool("postgres://hatchet:hatchet@localhost:5431/hatchet") as pool:
        yield {
            "foo": "bar",
            "pool": pool,
        }

    print("Cleaning up lifespan!")


@hatchet.task(name="LifespanWorkflow")
def lifespan_task(input: EmptyModel, ctx: Context) -> TaskOutput:
    pool = cast(ConnectionPool, ctx.lifespan["pool"])

    with pool.connection() as conn:
        query = conn.execute("SELECT * FROM v1_lookup_table_olap LIMIT 5;")
        rows = query.fetchall()

        for row in rows:
            print(row)

        print("executed step1 with lifespan", ctx.lifespan)

        return TaskOutput(
            num_rows=len(rows),
            external_ids=[cast(UUID, row[0]) for row in rows],
        )


def main() -> None:
    worker = hatchet.worker(
        "test-worker", slots=1, workflows=[lifespan_task], lifespan=lifespan
    )
    worker.start()


# ‼️

if __name__ == "__main__":
    main()

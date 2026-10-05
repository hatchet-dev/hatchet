from hatchet_sdk import Hatchet, TTLBasedIdempotencyConfig, EmptyModel, Context
from datetime import timedelta

hatchet = Hatchet()

durable_dag_idempotency_hang_bug_repro_wf = hatchet.workflow(
    name="durable-dag-idempotency-hang-bug-repro",
    idempotency=TTLBasedIdempotencyConfig(
        key_expression="'*'",
        ttl=timedelta(minutes=10),
    ),
)


@durable_dag_idempotency_hang_bug_repro_wf.task()
async def task_1(_i: EmptyModel, _c: Context) -> None:
    pass


@durable_dag_idempotency_hang_bug_repro_wf.task()
async def task_2(_i: EmptyModel, _c: Context) -> None:
    pass

from hatchet_sdk import (
    Hatchet,
    StatusBasedIdempotencyConfig,
    TTLBasedIdempotencyConfig,
    EmptyModel,
    Context,
)
from datetime import timedelta
from pydantic import BaseModel

hatchet = Hatchet()


class IdempotencyHangInput(BaseModel):
    key: str


durable_dag_idempotency_hang_bug_repro_wf = hatchet.workflow(
    name="durable-dag-idempotency-hang-bug-repro",
    idempotency=TTLBasedIdempotencyConfig(
        key_expression="input.key",
        ttl=timedelta(minutes=10),
    ),
    input_validator=IdempotencyHangInput,
)


@durable_dag_idempotency_hang_bug_repro_wf.task()
async def task_1(_i: IdempotencyHangInput, _c: Context) -> None:
    pass


@durable_dag_idempotency_hang_bug_repro_wf.task()
async def task_2(_i: IdempotencyHangInput, _c: Context) -> None:
    pass


durable_dag_status_based_idempotency_wf = hatchet.workflow(
    name="durable-dag-status-based-idempotency",
    idempotency=StatusBasedIdempotencyConfig(
        key_expression="input.key",
        fallback_ttl=timedelta(minutes=10),
    ),
    input_validator=IdempotencyHangInput,
)


@durable_dag_status_based_idempotency_wf.task()
async def status_based_task_1(_i: IdempotencyHangInput, _c: Context) -> None:
    pass


@durable_dag_status_based_idempotency_wf.task()
async def status_based_task_2(_i: IdempotencyHangInput, _c: Context) -> None:
    pass

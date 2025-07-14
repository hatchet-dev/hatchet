from pydantic import BaseModel

from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet


class UnitTestInput(BaseModel):
    key: str
    number: int


class UnitTestOutput(UnitTestInput):
    additional_metadata: dict[str, str]
    retry_count: int


class Lifespan(BaseModel):
    mock_db_url: str


hatchet = Hatchet()

## sync and async for top level task and durable
## sync and async for workflow task and durable
## sync and async tasks in dag
## 12 total


@hatchet.task(input_validator=UnitTestInput)
def sync_standalone(input: UnitTestInput, ctx: Context) -> UnitTestOutput:
    return UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=ctx.additional_metadata,
        retry_count=ctx.retry_count,
    )


@hatchet.task(input_validator=UnitTestInput)
async def async_standalone(input: UnitTestInput, ctx: Context) -> UnitTestOutput:
    return UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=ctx.additional_metadata,
        retry_count=ctx.retry_count,
    )


@hatchet.durable_task(input_validator=UnitTestInput)
def durable_sync_standalone(
    input: UnitTestInput, ctx: DurableContext
) -> UnitTestOutput:
    return UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=ctx.additional_metadata,
        retry_count=ctx.retry_count,
    )


@hatchet.durable_task(input_validator=UnitTestInput)
async def durable_async_standalone(
    input: UnitTestInput, ctx: DurableContext
) -> UnitTestOutput:
    return UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=ctx.additional_metadata,
        retry_count=ctx.retry_count,
    )


simple_workflow = hatchet.workflow(
    name="simple-unit-test-workflow", input_validator=UnitTestInput
)


@simple_workflow.task()
def sync_simple_workflow(input: UnitTestInput, ctx: Context) -> UnitTestOutput:
    return UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=ctx.additional_metadata,
        retry_count=ctx.retry_count,
    )


@simple_workflow.task()
async def async_simple_workflow(input: UnitTestInput, ctx: Context) -> UnitTestOutput:
    return UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=ctx.additional_metadata,
        retry_count=ctx.retry_count,
    )


@simple_workflow.durable_task()
def durable_sync_simple_workflow(
    input: UnitTestInput, ctx: DurableContext
) -> UnitTestOutput:
    return UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=ctx.additional_metadata,
        retry_count=ctx.retry_count,
    )


@simple_workflow.durable_task()
async def durable_async_simple_workflow(
    input: UnitTestInput, ctx: DurableContext
) -> UnitTestOutput:
    return UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=ctx.additional_metadata,
        retry_count=ctx.retry_count,
    )


complex_workflow = hatchet.workflow(
    name="complex-unit-test-workflow", input_validator=UnitTestInput
)


@complex_workflow.task()
async def start(input: UnitTestInput, ctx: Context) -> UnitTestOutput:
    return UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=ctx.additional_metadata,
        retry_count=ctx.retry_count,
    )


@complex_workflow.task(
    parents=[start],
)
def sync_complex_workflow(input: UnitTestInput, ctx: Context) -> UnitTestOutput:
    return ctx.task_output(start)


@complex_workflow.task(
    parents=[start],
)
async def async_complex_workflow(input: UnitTestInput, ctx: Context) -> UnitTestOutput:
    return ctx.task_output(start)


@complex_workflow.durable_task(
    parents=[start],
)
def durable_sync_complex_workflow(
    input: UnitTestInput, ctx: DurableContext
) -> UnitTestOutput:
    return ctx.task_output(start)


@complex_workflow.durable_task(
    parents=[start],
)
async def durable_async_complex_workflow(
    input: UnitTestInput, ctx: DurableContext
) -> UnitTestOutput:
    return ctx.task_output(start)

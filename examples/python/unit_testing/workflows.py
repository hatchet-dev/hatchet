from typing import cast

from pydantic import BaseModel

from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet


class UnitTestInput(BaseModel):
    key: str
    number: int


class Lifespan(BaseModel):
    mock_db_url: str


class UnitTestOutput(UnitTestInput, Lifespan):
    additional_metadata: dict[str, str]
    retry_count: int


hatchet = Hatchet()


@hatchet.task(input_validator=UnitTestInput)
def sync_standalone(input: UnitTestInput, ctx: Context) -> UnitTestOutput:
    return UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=ctx.additional_metadata,
        retry_count=ctx.retry_count,
        mock_db_url=cast(Lifespan, ctx.lifespan).mock_db_url,
    )


@hatchet.task(input_validator=UnitTestInput)
async def async_standalone(input: UnitTestInput, ctx: Context) -> UnitTestOutput:
    return UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=ctx.additional_metadata,
        retry_count=ctx.retry_count,
        mock_db_url=cast(Lifespan, ctx.lifespan).mock_db_url,
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
        mock_db_url=cast(Lifespan, ctx.lifespan).mock_db_url,
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
        mock_db_url=cast(Lifespan, ctx.lifespan).mock_db_url,
    )


@simple_workflow.task()
async def async_simple_workflow(input: UnitTestInput, ctx: Context) -> UnitTestOutput:
    return UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=ctx.additional_metadata,
        retry_count=ctx.retry_count,
        mock_db_url=cast(Lifespan, ctx.lifespan).mock_db_url,
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
        mock_db_url=cast(Lifespan, ctx.lifespan).mock_db_url,
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
        mock_db_url=cast(Lifespan, ctx.lifespan).mock_db_url,
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
async def durable_async_complex_workflow(
    input: UnitTestInput, ctx: DurableContext
) -> UnitTestOutput:
    return ctx.task_output(start)

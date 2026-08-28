import asyncio
from datetime import datetime, timedelta, timezone

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    DurableContext,
    EmptyModel,
    RateLimit,
    RateLimitDuration,
)

from .hatchet_client import hatchet


# > Hatchet task definition
class OrderInput(BaseModel):
    order_id: str


class ChargeOutput(BaseModel):
    charged: bool
    charge_id: str


@hatchet.task(name="charge-order", input_validator=OrderInput)
async def charge_order(input: OrderInput, ctx: Context) -> ChargeOutput:
    charge_id = await submit_charge(input.order_id)

    return ChargeOutput(charged=True, charge_id=charge_id)




class ValidateOutput(BaseModel):
    valid: bool


class FulfillOutput(BaseModel):
    order_id: str
    tracking_number: str


@hatchet.task(name="validate-order", input_validator=OrderInput)
async def validate_order(input: OrderInput, ctx: Context) -> ValidateOutput:
    return ValidateOutput(valid=await check_inventory(input.order_id))


@hatchet.task(name="fulfill-order", input_validator=OrderInput)
async def fulfill_order(input: OrderInput, ctx: Context) -> FulfillOutput:
    return FulfillOutput(
        order_id=input.order_id,
        tracking_number=await ship(input.order_id),
    )


# > Hatchet workflow as durable task
@hatchet.durable_task(name="ProcessOrder", input_validator=OrderInput)
async def process_order(input: OrderInput, ctx: DurableContext) -> FulfillOutput:
    await validate_order.aio_run(input)
    await charge_order.aio_run(input)

    return await fulfill_order.aio_run(input)




# > Hatchet task invocation
async def trigger_process_order(order_id: str) -> FulfillOutput:
    ref = await process_order.aio_run(
        OrderInput(order_id=order_id),
        wait_for_result=False,
    )

    # Available immediately. Store it if you need to reattach to the run later.
    print(ref.workflow_run_id)

    return await ref.aio_result()




# > Hatchet retries and timeouts
@hatchet.task(
    name="charge-order-with-retries",
    input_validator=OrderInput,
    retries=10,
    backoff_factor=2.0,
    backoff_max_seconds=10,
    execution_timeout=timedelta(seconds=30),
    schedule_timeout=timedelta(minutes=10),
)
async def charge_order_with_retries(input: OrderInput, ctx: Context) -> ChargeOutput:
    # Raising `NonRetryableException` here would stop Hatchet from retrying at all.
    if ctx.retry_count < 2:
        raise RuntimeError(f"payment provider unavailable for {input.order_id}")

    return ChargeOutput(charged=True, charge_id=await submit_charge(input.order_id))




class OnboardingInput(BaseModel):
    user_id: str


class EmailOutput(BaseModel):
    delivered: bool


@hatchet.task(name="send-welcome-email", input_validator=OnboardingInput)
async def send_welcome_email(input: OnboardingInput, ctx: Context) -> EmailOutput:
    return EmailOutput(delivered=await send_email(input.user_id, "welcome"))


@hatchet.task(name="send-followup-email", input_validator=OnboardingInput)
async def send_followup_email(input: OnboardingInput, ctx: Context) -> EmailOutput:
    return EmailOutput(delivered=await send_email(input.user_id, "followup"))


# > Hatchet durable task with sleep
@hatchet.durable_task(
    name="OnboardingFlow",
    input_validator=OnboardingInput,
    # The timeout has to cover the whole wall-clock span of the run, sleeps included.
    execution_timeout=timedelta(days=7),
)
async def onboarding_flow(input: OnboardingInput, ctx: DurableContext) -> None:
    await send_welcome_email.aio_run(input)

    await ctx.aio_sleep_for(timedelta(days=3))

    await send_followup_email.aio_run(input)




@hatchet.durable_task(name="WaitADay", execution_timeout=timedelta(days=2))
async def wait_a_day(input: EmptyModel, ctx: DurableContext) -> None:
    # > Hatchet durable sleep
    await ctx.aio_sleep_for(timedelta(days=1))


# > Hatchet event push
async def grant_approval(correlation_id: str) -> None:
    await hatchet.event.aio_push(
        "approval:granted",
        {"correlation_id": correlation_id},
    )




# > Hatchet durable event wait
class ApprovalInput(BaseModel):
    order_id: str
    # Generate this yourself, and keep it CEL-safe: the expression below is compiled
    # as-is, so a value containing a quote would fail to compile and the run would
    # wait until it timed out.
    correlation_id: str


class ApprovalOutput(BaseModel):
    approved: bool


@hatchet.durable_task(
    name="ApprovalFlow",
    input_validator=ApprovalInput,
    execution_timeout=timedelta(minutes=10),
)
async def approval_flow(input: ApprovalInput, ctx: DurableContext) -> ApprovalOutput:
    await ctx.aio_wait_for_event(
        "approval:granted",
        f"input.correlation_id == '{input.correlation_id}'",
    )

    return ApprovalOutput(approved=True)




# > Hatchet fan out children
class ItemInput(BaseModel):
    item_id: str


class ItemOutput(BaseModel):
    item_id: str
    result: str


class FanOutInput(BaseModel):
    item_ids: list[str]


class FanOutOutput(BaseModel):
    results: list[ItemOutput]


@hatchet.task(name="process-item", input_validator=ItemInput)
async def process_item(input: ItemInput, ctx: Context) -> ItemOutput:
    return ItemOutput(item_id=input.item_id, result=await handle_item(input.item_id))


@hatchet.durable_task(name="ProcessItems", input_validator=FanOutInput)
async def process_items(input: FanOutInput, ctx: DurableContext) -> FanOutOutput:
    results = await asyncio.gather(
        *[
            process_item.aio_run(ItemInput(item_id=item_id))
            for item_id in input.item_ids
        ]
    )

    return FanOutOutput(results=results)




# > Hatchet cron declaration
class ReportInput(BaseModel):
    kind: str


class ReportOutput(BaseModel):
    kind: str
    rows: int


@hatchet.task(
    name="weekly-report",
    input_validator=ReportInput,
    on_crons=["0 9 * * 1"],
    cron_input=ReportInput(kind="weekly"),
)
async def weekly_report(input: ReportInput, ctx: Context) -> ReportOutput:
    return ReportOutput(kind=input.kind, rows=await count_report_rows(input.kind))




# > Hatchet runtime schedules
async def create_schedules(customer_id: str) -> tuple[str, str]:
    cron = await hatchet.cron.aio_create(
        workflow_name=weekly_report.name,
        cron_name=f"weekly-report-{customer_id}",
        expression="0 9 * * 1",
        input={"kind": "weekly"},
        additional_metadata={"customer_id": customer_id},
    )

    scheduled = await hatchet.scheduled.aio_create(
        workflow_name=weekly_report.name,
        trigger_at=datetime.now(tz=timezone.utc) + timedelta(days=1),
        input={"kind": "weekly"},
        additional_metadata={"customer_id": customer_id},
    )

    return cron.metadata.id, scheduled.metadata.id




# > Hatchet concurrency and rate limits
class SyncInput(BaseModel):
    customer_id: str


class SyncOutput(BaseModel):
    records_synced: int


# One in-flight run per customer, newest cancels the oldest.
sync_customer = hatchet.workflow(
    name="SyncCustomer",
    input_validator=SyncInput,
    concurrency=ConcurrencyExpression(
        expression="input.customer_id",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
    ),
)


@sync_customer.task()
async def sync(input: SyncInput, ctx: Context) -> SyncOutput:
    return SyncOutput(records_synced=await sync_customer_data(input.customer_id))


class PromptInput(BaseModel):
    prompt: str


class ModelOutput(BaseModel):
    completion: str


# A global budget shared by every worker, not per-process. The static key has to be
# declared once with `hatchet.rate_limits.put` before a task can consume it.
@hatchet.task(
    name="call-model",
    input_validator=PromptInput,
    rate_limits=[RateLimit(static_key="openai", units=1)],
)
async def call_model(input: PromptInput, ctx: Context) -> ModelOutput:
    return ModelOutput(completion=await complete_prompt(input.prompt))




# > Hatchet DAG workflow
order_workflow = hatchet.workflow(name="ProcessOrderDag", input_validator=OrderInput)


@order_workflow.task(execution_timeout=timedelta(seconds=30))
async def validate(input: OrderInput, ctx: Context) -> ValidateOutput:
    return ValidateOutput(valid=await check_inventory(input.order_id))


@order_workflow.task(parents=[validate], execution_timeout=timedelta(seconds=30))
async def charge(input: OrderInput, ctx: Context) -> ChargeOutput:
    validated = ctx.task_output(validate)

    if not validated.valid:
        raise ValueError(f"order {input.order_id} failed validation")

    return ChargeOutput(charged=True, charge_id=await submit_charge(input.order_id))


@order_workflow.task(parents=[charge], execution_timeout=timedelta(seconds=30))
async def fulfill(input: OrderInput, ctx: Context) -> FulfillOutput:
    charged = ctx.task_output(charge)

    return FulfillOutput(
        order_id=input.order_id,
        tracking_number=await ship(charged.charge_id),
    )




@hatchet.task(name="charge-order-with-logging", input_validator=OrderInput)
async def charge_order_with_logging(input: OrderInput, ctx: Context) -> ChargeOutput:
    # > Hatchet context logging
    ctx.log("charging order")

    return ChargeOutput(charged=True, charge_id=await submit_charge(input.order_id))


# > Hatchet worker
def main() -> None:
    worker = hatchet.worker(
        "order-worker",
        slots=10,
        workflows=[validate_order, charge_order, fulfill_order, process_order],
    )
    worker.start()




# `main` above is the snippet the guide shows, so it registers only the four
# workflows that section is about. Running this file registers every example in
# it instead, which is what you want if you are trying them out.
def run_all_examples() -> None:
    # `call_model` consumes this static key, so it has to exist before the task runs.
    hatchet.rate_limits.put("openai", 1000, RateLimitDuration.MINUTE)

    worker = hatchet.worker(
        "temporal-migration-guide-worker",
        slots=10,
        workflows=[
            validate_order,
            charge_order,
            fulfill_order,
            process_order,
            charge_order_with_retries,
            charge_order_with_logging,
            send_welcome_email,
            send_followup_email,
            onboarding_flow,
            wait_a_day,
            approval_flow,
            process_item,
            process_items,
            weekly_report,
            sync_customer,
            call_model,
            order_workflow,
        ],
    )
    worker.start()


async def check_inventory(order_id: str) -> bool:
    return True


async def submit_charge(order_id: str) -> str:
    return f"ch_{order_id}"


async def ship(shipment_ref: str) -> str:
    return f"TRACK-{shipment_ref}"


async def send_email(user_id: str, template: str) -> bool:
    return True


async def handle_item(item_id: str) -> str:
    return f"processed-{item_id}"


async def count_report_rows(kind: str) -> int:
    return 7


async def sync_customer_data(customer_id: str) -> int:
    return 3


async def complete_prompt(prompt: str) -> str:
    return f"echo: {prompt}"


if __name__ == "__main__":
    run_all_examples()

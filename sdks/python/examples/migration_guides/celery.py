from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel

from .hatchet_client import hatchet

# --- Models used across snippets ---


class ImageInput(BaseModel):
    image_url: str
    filters: list[str]


class ImageOutput(BaseModel):
    processed_url: str


class OrderInput(BaseModel):
    order_id: str


class OrderValidated(BaseModel):
    order_id: str
    valid: bool


class ChargeResult(BaseModel):
    order_id: str
    charge_id: str


class FulfillResult(BaseModel):
    order_id: str
    tracking_number: str


class NotifyResult(BaseModel):
    order_id: str
    notified: bool


# --- Step 3: Task definition ---


# > Hatchet task definition
@hatchet.task(name="process-image", input_validator=ImageInput)
async def process_image(input: ImageInput, ctx: Context) -> ImageOutput:
    result = await resize(input.image_url, input.filters)
    return ImageOutput(processed_url=result)


# !!


# --- Step 4: Task invocation ---


# > Hatchet task invocation
async def run_image_task() -> None:
    # Wait for the result (default behavior)
    result = await process_image.aio_run(
        ImageInput(image_url="https://example.com/photo.png", filters=["thumbnail"]),
    )
    print(result.processed_url)

    # Fire-and-forget: enqueue without waiting
    ref = await process_image.aio_run(
        ImageInput(image_url="https://example.com/photo.png", filters=["thumbnail"]),
        wait_for_result=False,
    )
    print(ref.workflow_run_id)  # available immediately
    # await ref.aio_result() to retrieve the result later


# !!


# --- Step 5: Worker startup ---


# > Hatchet worker
def start_worker() -> None:
    worker = hatchet.worker("image-worker", slots=4, workflows=[process_image])
    worker.start()


# !!


# --- Step 6: Retries and timeouts ---


# > Hatchet retries
@hatchet.task(
    name="call-api",
    retries=5,
    backoff_factor=2.0,
    backoff_max_seconds=60,
    execution_timeout=timedelta(seconds=30),
    input_validator=OrderInput,
)
async def call_api(input: OrderInput, ctx: Context) -> dict[str, str]:
    result = await external_api_call(input.order_id)
    return {"status": result}


# !!


# --- Step 7a: Scheduled run (delayed execution) ---


# > Hatchet scheduled run
async def schedule_for_later() -> None:
    from datetime import datetime, timezone

    run_at = datetime.now(tz=timezone.utc) + timedelta(hours=1)
    await process_image.aio_schedule(
        run_at,
        ImageInput(image_url="https://example.com/photo.png", filters=["blur"]),
    )


# !!


# --- Step 7b: Cron (periodic tasks) ---


# > Hatchet cron
daily_report = hatchet.workflow(name="DailyReport", on_crons=["0 9 * * *"])


@daily_report.task()
async def generate_report(input: EmptyModel, ctx: Context) -> dict[str, str]:
    await build_report()
    return {"status": "sent"}


# !!


# --- Step 8: DAG workflow (chain/group/chord replacement) ---


# > Hatchet DAG workflow
order_pipeline = hatchet.workflow(name="OrderPipeline", input_validator=OrderInput)


@order_pipeline.task(execution_timeout=timedelta(seconds=30))
async def validate(input: OrderInput, ctx: Context) -> OrderValidated:
    ok = await check_inventory(input.order_id)
    return OrderValidated(order_id=input.order_id, valid=ok)


@order_pipeline.task(parents=[validate])
async def charge(input: OrderInput, ctx: Context) -> ChargeResult:
    parent = ctx.task_output(validate)
    cid = await process_charge(parent.order_id)
    return ChargeResult(order_id=input.order_id, charge_id=cid)


@order_pipeline.task(parents=[charge])
async def fulfill(input: OrderInput, ctx: Context) -> FulfillResult:
    parent = ctx.task_output(charge)
    tracking = await ship_order(parent.order_id)
    return FulfillResult(order_id=input.order_id, tracking_number=tracking)


@order_pipeline.task(parents=[fulfill])
async def notify(input: OrderInput, ctx: Context) -> NotifyResult:
    parent = ctx.task_output(fulfill)
    await send_notification(parent.order_id)
    return NotifyResult(order_id=input.order_id, notified=True)


# !!


# --- Stubs (not part of snippets) ---


async def resize(url: str, filters: list[str]) -> str:
    return f"https://cdn.example.com/{url.split('/')[-1]}"


async def external_api_call(order_id: str) -> str:
    return "ok"


async def build_report() -> str:
    return "report"


async def check_inventory(order_id: str) -> bool:
    return True


async def process_charge(order_id: str) -> str:
    return "ch_123"


async def ship_order(order_id: str) -> str:
    return "TRACK-456"


async def send_notification(order_id: str) -> None:
    pass

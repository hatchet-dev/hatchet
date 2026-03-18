"""
HatchetInstrumentor example — order processing workflow with custom spans.

Run the worker:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.worker

Then trigger it from another terminal:
    poetry run python -m examples.opentelemetry_instrumentation.hatchet.trigger
"""

import time

from pydantic import BaseModel

from opentelemetry.trace import StatusCode, get_tracer

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

# > Setup
HatchetInstrumentor().instrument()

hatchet = Hatchet()


class OrderInput(BaseModel):
    order_id: str
    customer_id: str
    amount: int


class ValidateOrderOutput(BaseModel):
    order_id: str
    valid: bool


class ChargePaymentOutput(BaseModel):
    transaction_id: str
    charged: int


class ReserveInventoryOutput(BaseModel):
    reservation_id: str
    items_reserved: int


class SendConfirmationOutput(BaseModel):
    sent: bool
    transaction_id: str
    reservation_id: str


otel_workflow = hatchet.workflow(
    name="otel-order-processing-py", input_validator=OrderInput
)


# > Custom Spans
@otel_workflow.task()
def validate_order(input: OrderInput, ctx: Context) -> ValidateOrderOutput:
    tracer = get_tracer(__name__)

    with tracer.start_as_current_span("order.validate.schema") as span:
        time.sleep(0.01)
        span.set_attribute("order.id", input.order_id)

    with tracer.start_as_current_span("order.validate.fraud-check") as span:
        time.sleep(0.02)
        span.set_attribute("fraud.score", 0.05)
        span.set_attribute("fraud.decision", "allow")

    return ValidateOrderOutput(order_id=input.order_id, valid=True)




@otel_workflow.task(parents=[validate_order])
def charge_payment(input: OrderInput, ctx: Context) -> ChargePaymentOutput:
    tracer = get_tracer(__name__)

    with tracer.start_as_current_span("payment.process") as pay_span:
        with tracer.start_as_current_span("payment.tokenize-card") as span:
            time.sleep(0.015)
            span.set_attribute("payment.provider", "stripe")

        with tracer.start_as_current_span("payment.charge") as span:
            time.sleep(0.03)
            span.set_attribute("payment.amount_cents", input.amount)
            span.set_attribute("payment.currency", "USD")

        pay_span.set_status(StatusCode.OK)

    return ChargePaymentOutput(
        transaction_id=f"txn-{input.order_id}",
        charged=input.amount,
    )


@otel_workflow.task(parents=[validate_order])
def reserve_inventory(input: OrderInput, ctx: Context) -> ReserveInventoryOutput:
    tracer = get_tracer(__name__)

    with tracer.start_as_current_span("inventory.check-availability") as span:
        time.sleep(0.01)
        span.set_attribute("inventory.sku_count", 3)
        span.set_attribute("inventory.all_available", True)

    with tracer.start_as_current_span("inventory.reserve") as span:
        time.sleep(0.015)
        span.set_attribute("inventory.warehouse", "us-east-1")

    return ReserveInventoryOutput(
        reservation_id=f"res-{input.order_id}",
        items_reserved=3,
    )


@otel_workflow.task(parents=[charge_payment, reserve_inventory])
def send_confirmation(input: OrderInput, ctx: Context) -> SendConfirmationOutput:
    tracer = get_tracer(__name__)

    with tracer.start_as_current_span("notification.render-template") as span:
        time.sleep(0.005)
        span.set_attribute("template.name", "order-confirmation")

    with tracer.start_as_current_span("notification.send-email") as span:
        time.sleep(0.02)
        span.set_attribute("email.to", "customer@example.com")
        span.set_attribute("email.provider", "sendgrid")

    return SendConfirmationOutput(
        sent=True,
        transaction_id=f"txn-{input.order_id}",
        reservation_id=f"res-{input.order_id}",
    )


# > Worker
def main() -> None:
    worker = hatchet.worker(
        "otel-instrumentation-worker-py",
        workflows=[otel_workflow],
    )
    worker.start()




if __name__ == "__main__":
    main()

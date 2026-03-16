"""
DAG workflow with OpenTelemetry instrumentation.

Run the worker:
    poetry run python -m examples.opentelemetry_dag.worker

Then trigger it from another terminal:
    poetry run python -m examples.opentelemetry_dag.trigger
"""

import time

from pydantic import BaseModel

from opentelemetry.trace import get_tracer

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

hatchet = Hatchet()

order_workflow = hatchet.workflow(name="OTelOrderProcessing")


class ValidateOrderOutput(BaseModel):
    valid: bool
    order_id: str


class ChargePaymentOutput(BaseModel):
    transaction_id: str
    charged: int


class ReserveInventoryOutput(BaseModel):
    reservation_id: str
    items_reserved: int


class SendConfirmationOutput(BaseModel):
    sent: bool


@order_workflow.task()
def validate_order(input: EmptyModel, ctx: Context) -> ValidateOrderOutput:
    tracer = get_tracer(__name__)

    with tracer.start_as_current_span(
        "order.validate.schema",
        attributes={"order.id": "order-123"},
    ):
        time.sleep(0.01)

    with tracer.start_as_current_span(
        "order.validate.fraud-check",
        attributes={"fraud.model": "v2.1"},
    ) as span:
        time.sleep(0.02)
        span.set_attribute("fraud.score", 0.05)
        span.set_attribute("fraud.decision", "allow")

    return ValidateOrderOutput(valid=True, order_id="order-123")


@order_workflow.task(parents=[validate_order])
def charge_payment(input: EmptyModel, ctx: Context) -> ChargePaymentOutput:
    tracer = get_tracer(__name__)

    with tracer.start_as_current_span("payment.process") as pay_span:
        with tracer.start_as_current_span("payment.tokenize-card") as span:
            time.sleep(0.015)
            span.set_attribute("payment.provider", "stripe")

        with tracer.start_as_current_span("payment.charge") as span:
            time.sleep(0.03)
            span.set_attribute("payment.amount_cents", 4999)
            span.set_attribute("payment.currency", "USD")

        pay_span.set_attribute("payment.success", True)

    return ChargePaymentOutput(transaction_id="txn-order-123", charged=4999)


@order_workflow.task(parents=[validate_order])
def reserve_inventory(input: EmptyModel, ctx: Context) -> ReserveInventoryOutput:
    tracer = get_tracer(__name__)

    with tracer.start_as_current_span("inventory.check-availability") as span:
        time.sleep(0.01)
        span.set_attribute("inventory.sku_count", 3)
        span.set_attribute("inventory.all_available", True)

    with tracer.start_as_current_span("inventory.reserve") as span:
        time.sleep(0.015)
        span.set_attribute("inventory.warehouse", "us-east-1")

    return ReserveInventoryOutput(reservation_id="res-order-123", items_reserved=3)


@order_workflow.task(parents=[charge_payment, reserve_inventory])
def send_confirmation(input: EmptyModel, ctx: Context) -> SendConfirmationOutput:
    tracer = get_tracer(__name__)

    with tracer.start_as_current_span("notification.render-template") as span:
        time.sleep(0.005)
        span.set_attribute("template.name", "order-confirmation")

    with tracer.start_as_current_span("notification.send-email") as span:
        time.sleep(0.02)
        span.set_attribute("email.to", "customer@example.com")
        span.set_attribute("email.provider", "sendgrid")

    return SendConfirmationOutput(sent=True)


def main() -> None:
    HatchetInstrumentor(enable_hatchet_otel_collector=True).instrument()

    worker = hatchet.worker(
        "otel-dag-worker",
        workflows=[order_workflow],
    )
    worker.start()


if __name__ == "__main__":
    main()

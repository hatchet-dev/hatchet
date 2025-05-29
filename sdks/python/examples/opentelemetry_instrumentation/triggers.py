import asyncio

from examples.opentelemetry_instrumentation.client import hatchet
from examples.opentelemetry_instrumentation.tracer import trace_provider
from examples.opentelemetry_instrumentation.worker import otel_workflow
from hatchet_sdk.clients.admin import TriggerWorkflowOptions
from hatchet_sdk.clients.events import BulkPushEventWithMetadata, PushEventOptions
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

instrumentor = HatchetInstrumentor(tracer_provider=trace_provider)
tracer = trace_provider.get_tracer(__name__)


def create_additional_metadata() -> dict[str, str]:
    return {"hello": "world"}


def create_push_options() -> PushEventOptions:
    return PushEventOptions(additional_metadata=create_additional_metadata())


def push_event() -> None:
    print("\npush_event")
    with tracer.start_as_current_span("push_event"):
        hatchet.event.push(
            "otel:event",
            {"test": "test"},
            options=create_push_options(),
        )


async def async_push_event() -> None:
    print("\nasync_push_event")
    with tracer.start_as_current_span("async_push_event"):
        await hatchet.event.aio_push(
            "otel:event", {"test": "test"}, options=create_push_options()
        )


def bulk_push_event() -> None:
    print("\nbulk_push_event")
    with tracer.start_as_current_span("bulk_push_event"):
        hatchet.event.bulk_push(
            [
                BulkPushEventWithMetadata(
                    key="otel:event",
                    payload={"test": "test 1"},
                    additional_metadata=create_additional_metadata(),
                ),
                BulkPushEventWithMetadata(
                    key="otel:event",
                    payload={"test": "test 2"},
                    additional_metadata=create_additional_metadata(),
                ),
            ],
        )


async def async_bulk_push_event() -> None:
    print("\nasync_bulk_push_event")
    with tracer.start_as_current_span("bulk_push_event"):
        await hatchet.event.aio_bulk_push(
            [
                BulkPushEventWithMetadata(
                    key="otel:event",
                    payload={"test": "test 1"},
                    additional_metadata=create_additional_metadata(),
                ),
                BulkPushEventWithMetadata(
                    key="otel:event",
                    payload={"test": "test 2"},
                    additional_metadata=create_additional_metadata(),
                ),
            ],
        )


def run_workflow() -> None:
    print("\nrun_workflow")
    with tracer.start_as_current_span("run_workflow"):
        otel_workflow.run(
            options=TriggerWorkflowOptions(
                additional_metadata=create_additional_metadata()
            ),
        )


async def async_run_workflow() -> None:
    print("\nasync_run_workflow")
    with tracer.start_as_current_span("async_run_workflow"):
        await otel_workflow.aio_run(
            options=TriggerWorkflowOptions(
                additional_metadata=create_additional_metadata()
            ),
        )


def run_workflows() -> None:
    print("\nrun_workflows")
    with tracer.start_as_current_span("run_workflows"):
        otel_workflow.run_many(
            [
                otel_workflow.create_bulk_run_item(
                    options=TriggerWorkflowOptions(
                        additional_metadata=create_additional_metadata()
                    )
                ),
                otel_workflow.create_bulk_run_item(
                    options=TriggerWorkflowOptions(
                        additional_metadata=create_additional_metadata()
                    )
                ),
            ],
        )


async def async_run_workflows() -> None:
    print("\nasync_run_workflows")
    with tracer.start_as_current_span("async_run_workflows"):
        await otel_workflow.aio_run_many(
            [
                otel_workflow.create_bulk_run_item(
                    options=TriggerWorkflowOptions(
                        additional_metadata=create_additional_metadata()
                    )
                ),
                otel_workflow.create_bulk_run_item(
                    options=TriggerWorkflowOptions(
                        additional_metadata=create_additional_metadata()
                    )
                ),
            ],
        )


async def main() -> None:
    push_event()
    await async_push_event()
    bulk_push_event()
    await async_bulk_push_event()
    run_workflow()
    # await async_run_workflow()
    run_workflows()
    # await async_run_workflows()


if __name__ == "__main__":
    asyncio.run(main())

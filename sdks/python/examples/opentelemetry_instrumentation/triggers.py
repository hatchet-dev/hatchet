import asyncio

from examples.opentelemetry_instrumentation.client import hatchet
from examples.opentelemetry_instrumentation.tracer import trace_provider
from hatchet_sdk.clients.admin import TriggerWorkflowOptions, WorkflowRunDict
from hatchet_sdk.clients.events import BulkPushEventWithMetadata, PushEventOptions
from hatchet_sdk.opentelemetry.instrumentor import (
    HatchetInstrumentor,
    inject_traceparent_into_metadata,
)

instrumentor = HatchetInstrumentor(tracer_provider=trace_provider)
tracer = trace_provider.get_tracer(__name__)


def create_additional_metadata() -> dict[str, str]:
    return inject_traceparent_into_metadata({"hello": "world"})


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
        hatchet.admin.run_workflow(
            "OTelWorkflow",
            {"test": "test"},
            options=TriggerWorkflowOptions(
                additional_metadata=create_additional_metadata()
            ),
        )


async def async_run_workflow() -> None:
    print("\nasync_run_workflow")
    with tracer.start_as_current_span("async_run_workflow"):
        await hatchet.admin.aio_run_workflow(
            "OTelWorkflow",
            {"test": "test"},
            options=TriggerWorkflowOptions(
                additional_metadata=create_additional_metadata()
            ),
        )


def run_workflows() -> None:
    print("\nrun_workflows")
    with tracer.start_as_current_span("run_workflows"):
        hatchet.admin.run_workflows(
            [
                WorkflowRunDict(
                    workflow_name="OTelWorkflow",
                    input={"test": "test"},
                    options=TriggerWorkflowOptions(
                        additional_metadata=create_additional_metadata()
                    ),
                ),
                WorkflowRunDict(
                    workflow_name="OTelWorkflow",
                    input={"test": "test 2"},
                    options=TriggerWorkflowOptions(
                        additional_metadata=create_additional_metadata()
                    ),
                ),
            ],
        )


async def async_run_workflows() -> None:
    print("\nasync_run_workflows")
    with tracer.start_as_current_span("async_run_workflows"):
        await hatchet.admin.aio_run_workflows(
            [
                WorkflowRunDict(
                    workflow_name="OTelWorkflow",
                    input={"test": "test"},
                    options=TriggerWorkflowOptions(
                        additional_metadata=create_additional_metadata()
                    ),
                ),
                WorkflowRunDict(
                    workflow_name="OTelWorkflow",
                    input={"test": "test 2"},
                    options=TriggerWorkflowOptions(
                        additional_metadata=create_additional_metadata()
                    ),
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

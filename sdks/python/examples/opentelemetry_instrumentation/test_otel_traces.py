import pytest
import asyncio
import time

from uuid import uuid4

from hatchet_sdk.clients.rest.models.otel_span import OtelSpan
from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from examples.opentelemetry_instrumentation.worker import (
    otel_simple_task,
    SimpleOtelTaskInput,
)
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor


def poll_for_trace(hatchet: Hatchet, run_id: str) -> list[OtelSpan]:
    for _ in range(10):
        with hatchet.runs.client() as client:
            trace = hatchet.runs._ta(client).v1_task_get_trace(run_id)

        spans = trace.rows or []
        if spans:
            return spans

        time.sleep(1)

    raise TimeoutError(f"Trace for run_id {run_id} not found after polling.")


@pytest.mark.asyncio(loop_scope="session")
async def test_otel_spans_created_on_task_run(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())
    message = "Hello, OpenTelemetry!"
    HatchetInstrumentor().instrument()

    ref = await otel_simple_task.aio_run_no_wait(
        input=SimpleOtelTaskInput(message=message),
        options=TriggerWorkflowOptions(
            additional_metadata={"test_run_id": test_run_id},
        ),
    )

    await ref.aio_result()

    spans = await asyncio.to_thread(poll_for_trace, hatchet, ref.workflow_run_id)
    step_run_spans = [s for s in spans if s.span_name == "hatchet.start_step_run"]
    assert len(step_run_spans) >= 1

    step_span = step_run_spans[0]
    attrs = step_span.span_attributes

    assert attrs

    assert attrs.get("hatchet.tenant_id") == hatchet.config.tenant_id
    assert attrs.get("hatchet.workflow_run_id") == ref.workflow_run_id
    assert attrs.get("hatchet.step_run_id") == ref.workflow_run_id
    assert attrs.get("hatchet.step_name") == otel_simple_task.name

    assert attrs.get("instrumentor") == "hatchet"

    child_spans = [s for s in spans if s.span_name == "custom.child.span"]
    assert len(child_spans) >= 1

    child_span = child_spans[0]
    child_attrs = child_span.span_attributes

    assert child_attrs

    assert child_attrs["hatchet.step_run_id"] == attrs["hatchet.step_run_id"]
    assert child_attrs.get("test.marker") == "hello"
    assert child_attrs.get("input.message") == message

    run_workflow_spans = [s for s in spans if s.span_name == "hatchet.run_workflow"]

    assert len(run_workflow_spans) == 1

    run_workflow_span = run_workflow_spans[0]

    assert run_workflow_span.span_attributes

    assert (
        run_workflow_span.span_attributes.get("hatchet.workflow_name")
        == otel_simple_task.name
    )

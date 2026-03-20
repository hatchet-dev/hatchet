import asyncio
from subprocess import Popen
from typing import Any

from uuid import uuid4
import pytest

from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from examples.opentelemetry_instrumentation.worker import (
    otel_retry_task,
    otel_simple_task,
)


@pytest.mark.asyncio(loop_scope="session")
async def test_otel_spans_created_on_task_run(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())

    ref = await otel_simple_task.aio_run_no_wait(
        options=TriggerWorkflowOptions(
            additional_metadata={"test_run_id": test_run_id},
        ),
    )

    await ref.aio_result()

    # Give the OTLP exporter time to flush spans to the collector
    await asyncio.sleep(10)

    with hatchet.runs.client() as client:
        trace = hatchet.runs._ta(client).v1_task_get_trace(ref.workflow_run_id)

    spans = trace.rows or []

    # Find the hatchet task run span
    step_run_spans = [s for s in spans if s.span_name == "hatchet.start_step_run"]
    assert len(step_run_spans) >= 1

    step_span = step_run_spans[0]
    attrs = step_span.span_attributes

    assert attrs

    tenant_id_attr = attrs.get("hatchet.tenant_id")
    workflow_run_id_attr = attrs.get("hatchet.workflow_run_id")
    step_run_id_attr = attrs.get("hatchet.step_run_id")
    step_name_attr = attrs.get("hatchet.step_name")

    assert tenant_id_attr == hatchet.config.tenant_id
    assert workflow_run_id_attr == ref.workflow_run_id
    assert step_run_id_attr == ref.workflow_run_id
    assert step_name_attr == otel_simple_task.name

    assert attrs.get("instrumentor") == "hatchet"

    child_spans = [s for s in spans if s.span_name == "custom.child.span"]
    assert len(child_spans) >= 1

    child_span = child_spans[0]
    child_attrs = child_span.span_attributes

    assert child_attrs

    assert child_attrs["hatchet.step_run_id"] == attrs["hatchet.step_run_id"]
    assert child_attrs.get("test.marker") == "hello"


# @pytest.mark.asyncio(loop_scope="session")
# async def test_otel_traces_on_retry(
#     hatchet: Hatchet
# ) -> None:
#     """Verify that traces are produced for both the failed attempt and the retry."""
#     ref = await otel_retry_task.aio_run_no_wait(
#         options=TriggerWorkflowOptions(
#             additional_metadata={"test_run_id": "otel-retry"},
#         ),
#     )

#     await ref.aio_result()

#     # Give the OTLP exporter time to flush spans to the collector
#     await asyncio.sleep(10)

#     spans = _get_trace_spans(hatchet, ref.workflow_run_id)

#     # The DB query returns only spans for the latest retry (MAX retry_count),
#     # so we expect exactly the successful retry's span.
#     step_run_spans = [s for s in spans if s.get("spanName") == "hatchet.start_step_run"]
#     assert len(step_run_spans) >= 1, (
#         f"Expected at least 1 hatchet task run span for the successful retry, "
#         f"got {len(step_run_spans)}. All spans: {[s.get('spanName') for s in spans]}"
#     )

#     # The returned span should be from the successful retry (retryCount >= 1)
#     latest_span = step_run_spans[0]
#     assert latest_span.get("retryCount", 0) >= 1, (
#         f"Expected span from retry attempt (retryCount >= 1), "
#         f"got retryCount={latest_span.get('retryCount')}"
#     )

#     # The span should have valid hatchet.* attributes
#     attrs = latest_span.get("spanAttributes", {})
#     assert "hatchet.step_run_id" in attrs, (
#         f"Step run span missing hatchet.step_run_id. Attrs: {attrs}"
#     )
#     assert "hatchet.workflow_run_id" in attrs, (
#         f"Step run span missing hatchet.workflow_run_id. Attrs: {attrs}"
#     )
#     assert "hatchet.tenant_id" in attrs, (
#         f"Step run span missing hatchet.tenant_id. Attrs: {attrs}"
#     )

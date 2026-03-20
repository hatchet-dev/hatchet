import asyncio
from subprocess import Popen
import time
from typing import Any

from uuid import uuid4
import pytest

from hatchet_sdk.clients.rest.models.otel_span import OtelSpan
from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from examples.opentelemetry_instrumentation.worker import (
    otel_retry_task,
    otel_simple_task,
)


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

    ref = await otel_simple_task.aio_run_no_wait(
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


@pytest.mark.asyncio(loop_scope="session")
async def test_otel_traces_on_retry(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())

    ref = await otel_retry_task.aio_run_no_wait(
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

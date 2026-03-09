import asyncio
from subprocess import Popen
from typing import Any
from uuid import uuid4

import pytest
import requests

from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from tests.otel_traces.worker import otel_retry_task, otel_simple_task

SPANS_PORT = 8020
WORKER_HEALTHCHECK_PORT = 8010

ON_DEMAND_WORKER_PARAMS = [
    (
        [
            "poetry",
            "run",
            "python",
            "tests/otel_traces/worker.py",
            "--spans-port",
            str(SPANS_PORT),
        ],
        WORKER_HEALTHCHECK_PORT,
    )
]


def _get_spans() -> list[dict[str, Any]]:
    """Fetch captured spans from the worker's span HTTP endpoint."""
    resp = requests.get(f"http://localhost:{SPANS_PORT}/spans", timeout=5)
    resp.raise_for_status()
    result: list[dict[str, Any]] = resp.json()
    return result


def _clear_spans() -> None:
    """Clear captured spans on the worker."""
    resp = requests.delete(f"http://localhost:{SPANS_PORT}/spans", timeout=5)
    resp.raise_for_status()


@pytest.mark.parametrize("on_demand_worker", ON_DEMAND_WORKER_PARAMS, indirect=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_otel_spans_created_on_task_run(
    hatchet: Hatchet, on_demand_worker: Popen[Any]
) -> None:
    """Verify that running a task produces correct OTel spans with hatchet.* attributes."""
    _clear_spans()

    test_run_id = str(uuid4())

    await otel_simple_task.aio_run(
        options=TriggerWorkflowOptions(
            additional_metadata={"test_run_id": test_run_id},
        ),
    )

    # Give the span processor a moment to flush
    await asyncio.sleep(1)

    spans = _get_spans()

    # Find the hatchet.start_step_run span
    step_run_spans = [s for s in spans if s["name"] == "hatchet.start_step_run"]
    assert len(step_run_spans) >= 1, (
        f"Expected at least one hatchet.start_step_run span, got {len(step_run_spans)}. "
        f"All spans: {[s['name'] for s in spans]}"
    )

    step_span = step_run_spans[-1]  # Use the most recent one

    # Verify hatchet attributes exist
    attrs = step_span["attributes"]
    assert "hatchet.step_run_id" in attrs, f"Missing hatchet.step_run_id in {attrs}"
    assert (
        "hatchet.workflow_run_id" in attrs
    ), f"Missing hatchet.workflow_run_id in {attrs}"
    assert "hatchet.tenant_id" in attrs, f"Missing hatchet.tenant_id in {attrs}"

    # Verify span kind is CONSUMER (value=4 in OTel Python SDK)
    assert step_span["kind"] == 4, f"Expected CONSUMER (4), got {step_span['kind']}"

    # Find the custom child span
    child_spans = [s for s in spans if s["name"] == "custom.child.span"]
    assert len(child_spans) >= 1, (
        f"Expected at least one custom.child.span, got {len(child_spans)}. "
        f"All spans: {[s['name'] for s in spans]}"
    )

    child_span = child_spans[-1]

    # Child span should share the same trace_id as the step run span
    assert (
        child_span["trace_id"] == step_span["trace_id"]
    ), f"Child trace_id {child_span['trace_id']} != step run trace_id {step_span['trace_id']}"

    # Child span should have hatchet.* attributes injected by _HatchetAttributeSpanProcessor
    child_attrs = child_span["attributes"]
    assert (
        "hatchet.step_run_id" in child_attrs
    ), f"Child span missing hatchet.step_run_id (attribute propagation failed). Attrs: {child_attrs}"
    assert (
        child_attrs["hatchet.step_run_id"] == attrs["hatchet.step_run_id"]
    ), "Child span hatchet.step_run_id doesn't match parent"

    # Verify the custom attribute is present
    assert (
        child_attrs.get("test.marker") == "hello"
    ), f"Missing test.marker attribute on child span. Attrs: {child_attrs}"


@pytest.mark.parametrize("on_demand_worker", ON_DEMAND_WORKER_PARAMS, indirect=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_otel_traces_on_retry(
    hatchet: Hatchet, on_demand_worker: Popen[Any]
) -> None:
    """Verify that traces are produced for both the failed attempt and the retry.

    Uses a task that fails on the first attempt (raises an exception) and
    succeeds on the second attempt (retries=1). Both attempts should produce
    valid hatchet.start_step_run spans with correct attributes, and the retry
    count attribute should differ between them.
    """
    _clear_spans()

    test_run_id = str(uuid4())

    await otel_retry_task.aio_run(
        options=TriggerWorkflowOptions(
            additional_metadata={"test_run_id": test_run_id},
        ),
    )

    # Give the span processor a moment to flush
    await asyncio.sleep(1)

    spans = _get_spans()

    # Both the failed first attempt and the successful retry should have spans
    step_run_spans = [s for s in spans if s["name"] == "hatchet.start_step_run"]
    assert len(step_run_spans) >= 2, (
        f"Expected at least 2 hatchet.start_step_run spans (initial + retry), "
        f"got {len(step_run_spans)}. All spans: {[s['name'] for s in spans]}"
    )

    # The first attempt should have errored
    error_spans = [s for s in step_run_spans if s["status_code"] == "ERROR"]
    assert len(error_spans) >= 1, (
        f"Expected at least one ERROR span from the failed first attempt. "
        f"Statuses: {[s['status_code'] for s in step_run_spans]}"
    )

    # All step run spans should have valid hatchet.* attributes
    for span in step_run_spans:
        attrs = span["attributes"]
        assert (
            "hatchet.step_run_id" in attrs
        ), f"Step run span missing hatchet.step_run_id. Attrs: {attrs}"
        assert (
            "hatchet.workflow_run_id" in attrs
        ), f"Step run span missing hatchet.workflow_run_id. Attrs: {attrs}"
        assert (
            "hatchet.tenant_id" in attrs
        ), f"Step run span missing hatchet.tenant_id. Attrs: {attrs}"

    # Verify retry count differs between attempts
    retry_counts = [s["attributes"].get("hatchet.retry_count") for s in step_run_spans]
    assert (
        len(set(retry_counts)) >= 2
    ), f"Expected different retry_count values across attempts, got {retry_counts}"

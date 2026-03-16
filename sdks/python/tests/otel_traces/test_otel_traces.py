import asyncio
from subprocess import Popen
from typing import Any

import pytest
import requests

from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from tests.otel_traces.worker import otel_retry_task, otel_simple_task

WORKER_HEALTHCHECK_PORT = 8010

ON_DEMAND_WORKER_PARAMS = [
    (
        ["poetry", "run", "python", "tests/otel_traces/worker.py"],
        WORKER_HEALTHCHECK_PORT,
    )
]


def _get_trace_spans(hatchet: Hatchet, workflow_run_id: str) -> list[dict[str, Any]]:
    """Fetch spans for a workflow run via the Hatchet REST API."""
    resp = requests.get(
        f"{hatchet.config.server_url}/api/v1/stable/workflow-runs/{workflow_run_id}/trace",
        headers={"Authorization": f"Bearer {hatchet.config.token}"},
        params={"limit": 1000},
        timeout=10,
    )
    resp.raise_for_status()
    data: dict[str, Any] = resp.json()
    result: list[dict[str, Any]] = data.get("rows", [])
    return result


@pytest.mark.parametrize("on_demand_worker", ON_DEMAND_WORKER_PARAMS, indirect=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_otel_spans_created_on_task_run(
    hatchet: Hatchet, on_demand_worker: Popen[Any]
) -> None:
    """Verify that running a task produces correct OTel spans with hatchet.* attributes."""
    ref = await otel_simple_task.aio_run_no_wait(
        options=TriggerWorkflowOptions(
            additional_metadata={"test_run_id": "otel-simple"},
        ),
    )

    await ref.aio_result()

    # Give the OTLP exporter time to flush spans to the collector
    await asyncio.sleep(3)

    spans = _get_trace_spans(hatchet, ref.workflow_run_id)

    # Find the hatchet task run span
    step_run_spans = [s for s in spans if s.get("spanName") == "hatchet.start_step_run"]
    assert len(step_run_spans) >= 1, (
        f"Expected at least one hatchet task run span, got {len(step_run_spans)}. "
        f"All spans: {[s.get('spanName') for s in spans]}"
    )

    step_span = step_run_spans[0]
    attrs = step_span.get("spanAttributes", {})

    # Verify hatchet attributes exist
    assert "hatchet.step_run_id" in attrs, f"Missing hatchet.step_run_id in {attrs}"
    assert (
        "hatchet.workflow_run_id" in attrs
    ), f"Missing hatchet.workflow_run_id in {attrs}"
    assert "hatchet.tenant_id" in attrs, f"Missing hatchet.tenant_id in {attrs}"

    # Verify instrumentor attribute
    assert (
        attrs.get("instrumentor") == "hatchet"
    ), f"Missing instrumentor attribute in {attrs}"

    # Find the custom child span
    child_spans = [s for s in spans if s.get("spanName") == "custom.child.span"]
    assert len(child_spans) >= 1, (
        f"Expected at least one custom.child.span, got {len(child_spans)}. "
        f"All spans: {[s.get('spanName') for s in spans]}"
    )

    child_span = child_spans[0]
    child_attrs = child_span.get("spanAttributes", {})

    # Child span should have hatchet.* attributes injected by _HatchetAttributeSpanProcessor
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
    """Verify that traces are produced for both the failed attempt and the retry."""
    ref = await otel_retry_task.aio_run_no_wait(
        options=TriggerWorkflowOptions(
            additional_metadata={"test_run_id": "otel-retry"},
        ),
    )

    await ref.aio_result()

    # Give the OTLP exporter time to flush spans to the collector
    await asyncio.sleep(3)

    spans = _get_trace_spans(hatchet, ref.workflow_run_id)

    # Both the failed first attempt and the successful retry should have spans
    step_run_spans = [s for s in spans if s.get("spanName") == "hatchet.start_step_run"]
    assert len(step_run_spans) >= 2, (
        f"Expected at least 2 hatchet task run spans (initial + retry), "
        f"got {len(step_run_spans)}. All spans: {[s.get('spanName') for s in spans]}"
    )

    # The first attempt should have errored
    error_spans = [s for s in step_run_spans if s.get("statusCode") == "ERROR"]
    assert len(error_spans) >= 1, (
        f"Expected at least one ERROR span from the failed first attempt. "
        f"Statuses: {[s.get('statusCode') for s in step_run_spans]}"
    )

    # All step run spans should have valid hatchet.* attributes
    for span in step_run_spans:
        attrs = span.get("spanAttributes", {})
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
    retry_counts = [
        s.get("spanAttributes", {}).get("hatchet.retry_count") for s in step_run_spans
    ]
    assert (
        len(set(retry_counts)) >= 2
    ), f"Expected different retry_count values across attempts, got {retry_counts}"

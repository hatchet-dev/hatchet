import asyncio
import os
import shutil
import subprocess
import time
from subprocess import Popen
from typing import Any
from uuid import uuid4

import pytest
import requests

from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from tests.otel_traces.worker import otel_long_task, otel_simple_task

SPANS_PORT = 8020
WORKER_HEALTHCHECK_PORT = 8010


def _get_spans() -> list[dict[str, Any]]:
    """Fetch captured spans from the worker's span HTTP endpoint."""
    resp = requests.get(f"http://localhost:{SPANS_PORT}/spans", timeout=5)
    resp.raise_for_status()
    return resp.json()


def _clear_spans() -> None:
    """Clear captured spans on the worker."""
    resp = requests.delete(f"http://localhost:{SPANS_PORT}/spans", timeout=5)
    resp.raise_for_status()


@pytest.mark.parametrize(
    "on_demand_worker",
    [
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
    ],
    indirect=True,
)
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
    assert "hatchet.workflow_run_id" in attrs, (
        f"Missing hatchet.workflow_run_id in {attrs}"
    )
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
    assert child_span["trace_id"] == step_span["trace_id"], (
        f"Child trace_id {child_span['trace_id']} != step run trace_id {step_span['trace_id']}"
    )

    # Child span should have hatchet.* attributes injected by _HatchetAttributeSpanProcessor
    child_attrs = child_span["attributes"]
    assert "hatchet.step_run_id" in child_attrs, (
        f"Child span missing hatchet.step_run_id (attribute propagation failed). Attrs: {child_attrs}"
    )
    assert child_attrs["hatchet.step_run_id"] == attrs["hatchet.step_run_id"], (
        "Child span hatchet.step_run_id doesn't match parent"
    )

    # Verify the custom attribute is present
    assert child_attrs.get("test.marker") == "hello", (
        f"Missing test.marker attribute on child span. Attrs: {child_attrs}"
    )


def _find_engine_pids() -> list[int]:
    """Find PIDs of running hatchet-engine processes."""
    try:
        result = subprocess.run(
            ["pgrep", "-f", "hatchet-engine"],
            capture_output=True,
            text=True,
        )
        if result.returncode == 0:
            return [int(pid) for pid in result.stdout.strip().split("\n") if pid]
    except Exception:
        pass
    return []


def _kill_engine() -> bool:
    """Kill hatchet-engine processes. Returns True if any were found and killed."""
    pids = _find_engine_pids()
    if not pids:
        return False
    subprocess.run(["pkill", "-f", "hatchet-engine"], capture_output=True)
    return True


def _restart_engine() -> bool:
    """Restart the hatchet-engine process. Returns True if successful.

    Set HATCHET_ENGINE_RESTART_CMD to a shell command that starts the engine,
    e.g. 'cd /path/to/repo && go run ./cmd/hatchet-engine --config ./generated/'
    or 'docker compose -f /path/to/docker-compose.yml up -d hatchet-engine'.

    Falls back to 'go run ./cmd/hatchet-engine --config ./generated/' from the repo root.
    """
    custom_cmd = os.environ.get("HATCHET_ENGINE_RESTART_CMD")
    if custom_cmd:
        subprocess.Popen(custom_cmd, shell=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)  # noqa: S602
        return True

    # Auto-detect repo root by walking up from this file
    # tests/otel_traces/test_otel_traces.py -> tests/ -> sdks/python/ -> sdks/ -> repo root
    sdk_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    repo_root = os.path.dirname(os.path.dirname(sdk_dir))

    config_dir = os.path.join(repo_root, "generated")
    if not os.path.isdir(config_dir):
        return False

    go_bin = shutil.which("go")
    if not go_bin:
        return False

    subprocess.Popen(
        [go_bin, "run", "./cmd/hatchet-engine", "--config", "./generated/"],
        cwd=repo_root,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    return True


@pytest.mark.parametrize(
    "on_demand_worker",
    [
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
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_otel_traces_after_engine_restart(
    hatchet: Hatchet, on_demand_worker: Popen[Any]
) -> None:
    """Verify traces work after the engine is killed and restarted mid-run.

    The worker should reconnect, the task should be retried, and the retried
    run should produce valid OTel spans.
    """
    # Skip if no engine process is found (not running in the expected environment)
    engine_pids = _find_engine_pids()
    if not engine_pids:
        pytest.skip("No hatchet-engine process found — skipping engine restart test")

    _clear_spans()

    test_run_id = str(uuid4())

    # Trigger a long-running task without waiting
    run_ref = await otel_long_task.aio_run_no_wait(
        options=TriggerWorkflowOptions(
            additional_metadata={"test_run_id": test_run_id},
        ),
    )

    # Wait for the task to start (poll for hatchet.start_step_run span)
    for _ in range(10):
        await asyncio.sleep(1)
        spans = _get_spans()
        step_run_spans = [s for s in spans if s["name"] == "hatchet.start_step_run"]
        if step_run_spans:
            break
    else:
        pytest.fail("Task did not start within 10 seconds (no hatchet.start_step_run span)")

    # Kill the engine
    killed = _kill_engine()
    assert killed, "Failed to kill hatchet-engine process"

    # Wait for disconnect to be detected
    await asyncio.sleep(3)

    # Restart the engine
    restarted = _restart_engine()
    if not restarted:
        pytest.skip(
            "Could not restart hatchet-engine (no generated/ config dir found). "
            "Set HATCHET_ENGINE_RESTART_CMD env var to a shell command that starts the engine."
        )

    # Wait for engine to come back up and worker to reconnect
    await asyncio.sleep(15)

    # Poll for the workflow run to reach a terminal state
    terminal_statuses = {V1TaskStatus.COMPLETED, V1TaskStatus.FAILED, V1TaskStatus.CANCELLED}
    final_status = None

    for _ in range(30):
        runs = await otel_long_task.aio_list_runs(
            additional_metadata={"test_run_id": test_run_id},
        )
        if runs:
            run = runs[0]
            if run.status in terminal_statuses:
                final_status = run.status
                break
        await asyncio.sleep(2)

    # Verify the worker subprocess is still alive (didn't crash)
    assert on_demand_worker.poll() is None, "Worker process crashed during engine restart"

    # Query final spans
    spans = _get_spans()
    step_run_spans = [s for s in spans if s["name"] == "hatchet.start_step_run"]

    # At least one step run span should exist (could be 2+ if retried)
    assert len(step_run_spans) >= 1, (
        f"Expected at least one hatchet.start_step_run span after engine restart. "
        f"All spans: {[s['name'] for s in spans]}"
    )

    # All step run spans should have valid hatchet.* attributes
    for span in step_run_spans:
        attrs = span["attributes"]
        assert "hatchet.step_run_id" in attrs, (
            f"Step run span missing hatchet.step_run_id after restart. Attrs: {attrs}"
        )
        assert "hatchet.workflow_run_id" in attrs, (
            f"Step run span missing hatchet.workflow_run_id after restart. Attrs: {attrs}"
        )
        assert "hatchet.tenant_id" in attrs, (
            f"Step run span missing hatchet.tenant_id after restart. Attrs: {attrs}"
        )

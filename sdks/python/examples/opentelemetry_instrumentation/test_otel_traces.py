import pytest
import asyncio
import time

from uuid import uuid4

from hatchet_sdk.clients.rest.models.otel_span import OtelSpan
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from hatchet_sdk.clients.events import PushEventOptions
from examples.opentelemetry_instrumentation.worker import (
    otel_simple_task,
    otel_spawn_parent,
    otel_workflow,
    SimpleOtelTaskInput,
)
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

requires_observability = pytest.mark.usefixtures("_skip_unless_observability")


def poll_for_trace(hatchet: Hatchet, run_id: str, min_spans: int = 1) -> list[OtelSpan]:
    # sleep to avoid race conditions with engine spans
    time.sleep(5)

    for _ in range(10):
        with hatchet.runs.client() as client:
            try:
                trace = hatchet.runs._wra(client).v1_workflow_run_get_trace(
                    hatchet.tenant_id, run_id
                )
            except Exception:
                time.sleep(1)
                continue

        spans = trace.rows or []
        if len(spans) >= min_spans:
            return spans

        time.sleep(1)

    raise TimeoutError(f"Trace for run_id {run_id} not found after polling.")


@requires_observability
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

    assert (
        len({s.trace_id for s in spans}) == 1
    ), "All spans should have the same trace_id"
    assert len({s.span_id for s in spans}) == len(
        spans
    ), "All spans should have unique span_ids"

    assert (
        len(spans) == 5
    ), "five spans: hatchet.run_workflow, hatchet.engine.queued,hatchet.start_step_run, hatchet.engine.start_step_run, custom.child.span"

    assert {s.span_name for s in spans} == {
        "hatchet.run_workflow",
        "hatchet.engine.queued",
        "hatchet.start_step_run",
        "hatchet.engine.start_step_run",
        "custom.child.span",
    }

    step_run_spans = [s for s in spans if s.span_name == "hatchet.start_step_run"]
    assert len(step_run_spans) >= 1

    sdk_spans = [
        s
        for s in step_run_spans
        if s.span_attributes and s.span_attributes.get("instrumentor") == "hatchet"
    ]
    assert len(sdk_spans) >= 1

    step_span = sdk_spans[0]
    attrs = step_span.span_attributes

    assert attrs

    assert attrs.get("hatchet.tenant_id") == hatchet.config.tenant_id
    assert attrs.get("hatchet.workflow_run_id") == ref.workflow_run_id
    assert attrs.get("hatchet.step_run_id") == ref.workflow_run_id
    assert (
        hatchet.config.apply_namespace(attrs.get("hatchet.step_name"))
        == otel_simple_task.name
    )

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
        hatchet.config.apply_namespace(
            run_workflow_span.span_attributes.get("hatchet.workflow_name")
        )
        == otel_simple_task.name
    )


@requires_observability
@pytest.mark.asyncio(loop_scope="session")
async def test_otel_spans_on_event_triggered_run(hatchet: Hatchet) -> None:
    HatchetInstrumentor().instrument()
    test_run_id = str(uuid4())

    event = await hatchet.event.aio_push(
        "otel:test-event",
        {"message": "event-triggered"},
        options=PushEventOptions(additional_metadata={"test_run_id": test_run_id}),
    )

    run_id = None
    for _ in range(15):
        runs = await hatchet.runs.aio_list(triggering_event_external_id=event.event_id)
        rows = runs.rows or []
        completed = [r for r in rows if r.status == V1TaskStatus.COMPLETED]
        if completed:
            run_id = completed[0].task_external_id
            break
        await asyncio.sleep(1)

    assert run_id is not None, "Event-triggered run did not complete in time."

    spans = await asyncio.to_thread(poll_for_trace, hatchet, run_id)

    assert (
        len(spans) == 5
    ), "five spans: hatchet.push_event, hatchet.engine.queued, hatchet.start_step_run, hatchet.engine.start_step_run, custom.child.span"

    assert (
        len({s.trace_id for s in spans}) == 1
    ), "All spans should have the same trace_id"
    assert len({s.span_id for s in spans}) == len(
        spans
    ), "All spans should have unique span_ids"

    assert {s.span_name for s in spans} == {
        "hatchet.push_event",
        "hatchet.engine.queued",
        "hatchet.start_step_run",
        "custom.child.span",
        "hatchet.engine.start_step_run",
    }

    push_event_spans = [s for s in spans if s.span_name == "hatchet.push_event"]

    assert len(push_event_spans) == 1

    push_event_span = push_event_spans[0]

    assert push_event_span.span_attributes
    assert push_event_span.span_attributes.get("hatchet.event_key") == "otel:test-event"

    step_run_spans = [s for s in spans if s.span_name == "hatchet.start_step_run"]
    assert len(step_run_spans) >= 1

    sdk_spans = [
        s
        for s in step_run_spans
        if s.span_attributes and s.span_attributes.get("instrumentor") == "hatchet"
    ]
    assert len(sdk_spans) >= 1

    attrs = sdk_spans[0].span_attributes
    assert attrs
    assert attrs.get("hatchet.tenant_id") == hatchet.config.tenant_id
    assert (
        hatchet.config.apply_namespace(attrs.get("hatchet.step_name"))
        == otel_simple_task.name
    )

    child_spans = [s for s in spans if s.span_name == "custom.child.span"]
    assert len(child_spans) >= 1
    assert child_spans[0].span_attributes
    assert child_spans[0].span_attributes.get("input.message") == "event-triggered"


@requires_observability
@pytest.mark.asyncio(loop_scope="session")
async def test_otel_spans_on_dag_run(hatchet: Hatchet) -> None:
    HatchetInstrumentor().instrument()

    ref = await otel_workflow.aio_run_no_wait()
    await ref.aio_result()

    spans = await asyncio.to_thread(
        poll_for_trace, hatchet, ref.workflow_run_id, min_spans=4
    )

    assert (
        len({s.trace_id for s in spans}) == 1
    ), "All spans should have the same trace_id"
    assert len({s.span_id for s in spans}) == len(
        spans
    ), "All spans should have unique span_ids"

    assert len(spans) == 24, """
        24 spans:
            - hatchet.run_workflow
            - 4x hatchet.engine.queued
            - hatchet.start_step_run for each of the 4 tasks
            - hatchet.engine.start_step_run for each of the 4 tasks
            - db.query
            - transform.pipeline
            - transform.normalize
            - http.request
            - schema.validate
            - transform.enrich
            - data.clean
            - transform.aggregate
            - cache.invalidate
            - notification.send
            - json.parse
        """

    assert {s.span_name for s in spans} == {
        "hatchet.run_workflow",
        "hatchet.engine.queued",
        "hatchet.start_step_run",
        "db.query",
        "transform.pipeline",
        "transform.normalize",
        "http.request",
        "schema.validate",
        "transform.enrich",
        "data.clean",
        "transform.aggregate",
        "cache.invalidate",
        "notification.send",
        "json.parse",
        "hatchet.engine.start_step_run",
    }

    step_run_spans = [s for s in spans if s.span_name == "hatchet.start_step_run"]
    step_names = {
        s.span_attributes.get("hatchet.step_name")
        for s in step_run_spans
        if s.span_attributes
    }

    expected_steps = {t.name for t in otel_workflow.tasks}
    assert expected_steps <= step_names

    sdk_spans = [
        s
        for s in step_run_spans
        if s.span_attributes and s.span_attributes.get("instrumentor") == "hatchet"
    ]
    assert len(sdk_spans) >= 1

    for span in sdk_spans:
        assert span.span_attributes is not None
        assert (
            span.span_attributes.get("hatchet.workflow_run_id") == ref.workflow_run_id
        )

    user_span_names = {s.span_name for s in spans}
    assert "http.request" in user_span_names
    assert "schema.validate" in user_span_names
    assert "transform.pipeline" in user_span_names
    assert "db.query" in user_span_names

    run_workflow_spans = [s for s in spans if s.span_name == "hatchet.run_workflow"]
    assert len(run_workflow_spans) == 1
    assert run_workflow_spans[0].span_attributes
    assert (
        hatchet.config.apply_namespace(
            run_workflow_spans[0].span_attributes.get("hatchet.workflow_name")
        )
        == otel_workflow.name
    )


@requires_observability
@pytest.mark.asyncio(loop_scope="session")
async def test_otel_spans_on_child_spawn(hatchet: Hatchet) -> None:
    HatchetInstrumentor().instrument()
    message = "spawn-test"
    test_run_id = str(uuid4())

    ref = await otel_spawn_parent.aio_run_no_wait(
        input=SimpleOtelTaskInput(message=message),
        options=TriggerWorkflowOptions(
            additional_metadata={"test_run_id": test_run_id},
        ),
    )

    await ref.aio_result()

    spans = await asyncio.to_thread(poll_for_trace, hatchet, ref.workflow_run_id)

    assert len(spans) == 10, """
        10 spans:
            - 2x hatchet.run_workflow (one for parent, one for child)
            - 2x hatchet.engine.queued (one for parent, one for child)
            - 2x hatchet.start_step_run for parent and child
            - 2x hatchet.engine.start_step_run for parent and child
            - spawn.child
            - custom.child.span
    """

    assert (
        len({s.trace_id for s in spans}) == 1
    ), "All spans should have the same trace_id"
    assert len({s.span_id for s in spans}) == len(
        spans
    ), "All spans should have unique span_ids"

    assert {s.span_name for s in spans} == {
        "hatchet.run_workflow",
        "hatchet.engine.queued",
        "hatchet.start_step_run",
        "spawn.child",
        "custom.child.span",
        "hatchet.engine.start_step_run",
    }

    step_run_spans = [s for s in spans if s.span_name == "hatchet.start_step_run"]
    assert len(step_run_spans) >= 1

    parent_span = step_run_spans[0]
    assert parent_span.span_attributes
    assert (
        hatchet.config.apply_namespace(
            parent_span.span_attributes.get("hatchet.step_name")
        )
        == otel_spawn_parent.name
    )

    spawn_spans = [s for s in spans if s.span_name == "spawn.child"]
    assert len(spawn_spans) >= 1
    assert spawn_spans[0].span_attributes
    assert spawn_spans[0].span_attributes.get("parent.message") == message

    run_workflow_spans = [s for s in spans if s.span_name == "hatchet.run_workflow"]
    assert len(run_workflow_spans) >= 1

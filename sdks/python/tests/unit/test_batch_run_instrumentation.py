"""Tests for batch run_workflows instrumentation:
- creates one hatchet.run_workflow span per item under the parent hatchet.run_workflows span
- injects each item's own traceparent into its additional_metadata
- ends all started item spans (with error status) when attribute building
  or the underlying wrapped call fails synchronously
"""

from __future__ import annotations

from collections.abc import Iterator
from unittest.mock import MagicMock

import pytest
from opentelemetry.sdk.trace import ReadableSpan, TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.sdk.trace.export.in_memory_span_exporter import (
    InMemorySpanExporter,
)
from opentelemetry.trace import StatusCode, get_tracer

from hatchet_sdk.opentelemetry.instrumentor import (
    HatchetInstrumentor,
    _create_traceparent_from_span,
)
from hatchet_sdk.types.trigger import TriggerWorkflowOptions, WorkflowRunTriggerConfig

BATCH_SPAN_NAME = "hatchet.run_workflows"
ITEM_SPAN_NAME = "hatchet.run_workflow"


def _make_config(name: str) -> WorkflowRunTriggerConfig:
    return WorkflowRunTriggerConfig(
        workflow_name=name,
        input=None,
        options=TriggerWorkflowOptions(additional_metadata={"caller": "test"}),
    )


def _spans_by_name(exporter: InMemorySpanExporter, name: str) -> list[ReadableSpan]:
    return [s for s in exporter.get_finished_spans() if s.name == name]


@pytest.fixture
def exporter() -> Iterator[InMemorySpanExporter]:
    provider = TracerProvider()
    exp = InMemorySpanExporter()
    provider.add_span_processor(SimpleSpanProcessor(exp))
    yield exp
    provider.shutdown()


@pytest.fixture
def instrumentor(exporter: InMemorySpanExporter) -> HatchetInstrumentor:
    # ClientConfig requires a JWT token. Bypass __init__ since we only need
    # the tracer and config.otel.excluded_attributes for the wrappers.
    inst = object.__new__(HatchetInstrumentor)
    provider = TracerProvider()
    provider.add_span_processor(SimpleSpanProcessor(exporter))
    inst._tracer = get_tracer(__name__, "test", provider)  # type: ignore[attr-defined]
    inst.config = MagicMock()
    inst.config.otel.excluded_attributes = []
    return inst


def test_run_workflows_creates_one_item_span_per_config_with_unique_traceparent(
    instrumentor: HatchetInstrumentor, exporter: InMemorySpanExporter
) -> None:
    captured: list[WorkflowRunTriggerConfig] = []

    def wrapped(configs: list[WorkflowRunTriggerConfig]) -> list[None]:
        captured.extend(configs)
        return [None for _ in configs]

    configs = [_make_config(f"wf-{i}") for i in range(3)]
    instrumentor._wrap_run_workflows(wrapped, MagicMock(), (configs,), {})

    batch_spans = _spans_by_name(exporter, BATCH_SPAN_NAME)
    item_spans = _spans_by_name(exporter, ITEM_SPAN_NAME)
    assert len(batch_spans) == 1
    assert len(item_spans) == 3

    batch_span_id = batch_spans[0].context.span_id
    for s in item_spans:
        assert s.parent is not None
        assert s.parent.span_id == batch_span_id
        assert s.status.status_code == StatusCode.UNSET

    # Each enhanced config carries a traceparent matching ITS OWN item span,
    # not the batch span — this is the regression the PR fixes.
    traceparents = [c.options.additional_metadata["traceparent"] for c in captured]
    assert len(set(traceparents)) == 3
    item_span_ids = {format(s.context.span_id, "016x") for s in item_spans}
    for tp in traceparents:
        # traceparent format: 00-<trace_id>-<span_id>-<flags>
        span_id_hex = tp.split("-")[2]
        assert span_id_hex in item_span_ids
        assert span_id_hex != format(batch_span_id, "016x")


async def test_async_run_workflows_creates_one_item_span_per_config(
    instrumentor: HatchetInstrumentor, exporter: InMemorySpanExporter
) -> None:
    async def wrapped(configs: list[WorkflowRunTriggerConfig]) -> list[None]:
        return [None for _ in configs]

    configs = [_make_config(f"wf-{i}") for i in range(2)]
    await instrumentor._wrap_async_run_workflows(wrapped, MagicMock(), (configs,), {})

    assert len(_spans_by_name(exporter, BATCH_SPAN_NAME)) == 1
    assert len(_spans_by_name(exporter, ITEM_SPAN_NAME)) == 2


def test_run_workflows_ends_spans_with_error_when_attribute_build_raises(
    instrumentor: HatchetInstrumentor, exporter: InMemorySpanExporter
) -> None:
    boom = RuntimeError("attribute boom")
    call_count = {"n": 0}
    original_build = instrumentor._build_run_workflow_attributes

    def build_with_failure(
        config: WorkflowRunTriggerConfig,
    ) -> dict[str, object]:
        call_count["n"] += 1
        if call_count["n"] == 2:
            raise boom
        return original_build(config)

    instrumentor._build_run_workflow_attributes = build_with_failure  # type: ignore[assignment]

    wrapped_calls: list[list[WorkflowRunTriggerConfig]] = []

    def wrapped(configs: list[WorkflowRunTriggerConfig]) -> list[None]:
        wrapped_calls.append(configs)
        return [None for _ in configs]

    configs = [_make_config(f"wf-{i}") for i in range(3)]

    with pytest.raises(RuntimeError, match="attribute boom"):
        instrumentor._wrap_run_workflows(wrapped, MagicMock(), (configs,), {})

    assert wrapped_calls == []
    item_spans = _spans_by_name(exporter, ITEM_SPAN_NAME)
    # The first item span was started successfully; the second failed before
    # start_span returned. The first must still be ended with error status.
    assert len(item_spans) == 1
    assert item_spans[0].status.status_code == StatusCode.ERROR


def test_run_workflows_ends_spans_with_error_when_wrapped_raises_synchronously(
    instrumentor: HatchetInstrumentor, exporter: InMemorySpanExporter
) -> None:
    boom = RuntimeError("sync wrapped boom")

    def wrapped(configs: list[WorkflowRunTriggerConfig]) -> list[None]:
        raise boom

    configs = [_make_config(f"wf-{i}") for i in range(2)]

    with pytest.raises(RuntimeError, match="sync wrapped boom"):
        instrumentor._wrap_run_workflows(wrapped, MagicMock(), (configs,), {})

    item_spans = _spans_by_name(exporter, ITEM_SPAN_NAME)
    assert len(item_spans) == 2
    for s in item_spans:
        assert s.status.status_code == StatusCode.ERROR


async def test_async_run_workflows_ends_spans_with_error_when_wrapped_raises(
    instrumentor: HatchetInstrumentor, exporter: InMemorySpanExporter
) -> None:
    boom = RuntimeError("async wrapped boom")

    async def wrapped(configs: list[WorkflowRunTriggerConfig]) -> list[None]:
        raise boom

    configs = [_make_config(f"wf-{i}") for i in range(2)]

    with pytest.raises(RuntimeError, match="async wrapped boom"):
        await instrumentor._wrap_async_run_workflows(
            wrapped, MagicMock(), (configs,), {}
        )

    item_spans = _spans_by_name(exporter, ITEM_SPAN_NAME)
    assert len(item_spans) == 2
    for s in item_spans:
        assert s.status.status_code == StatusCode.ERROR

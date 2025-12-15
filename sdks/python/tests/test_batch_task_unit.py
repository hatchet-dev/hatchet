import asyncio
import queue
from dataclasses import dataclass
from typing import Any

import pytest
from pydantic import BaseModel

from hatchet_sdk.config import ClientConfig
from hatchet_sdk.hatchet import Hatchet
from hatchet_sdk.worker.runner.runner import Runner


@dataclass
class FakeCtx:
    workflow_input: dict[str, Any]


@pytest.mark.asyncio(loop_scope="session")
async def test_batch_flush_orders_by_index() -> None:
    class Inp(BaseModel):
        value: str

    hatchet = Hatchet(config=ClientConfig())
    wf = hatchet.workflow(name="batch-unit-test", input_validator=Inp)

    @wf.batch_task(name="step", batch_size=3)
    async def step(inputs: list[Inp], ctxs: list[Any]) -> list[dict[str, str]]:
        return [{"uppercase": i.value.upper()} for i in inputs]

    action_id = wf._create_action_name(step)
    runner = Runner(
        event_queue=queue.Queue(),
        config=hatchet.config,
        slots=10,
        handle_kill=False,
        action_registry={action_id: step},
        labels=None,
        lifespan_context=None,
        log_sender=object(),  # not used in these unit tests
    )

    controller = runner._batch_controllers[action_id]
    loop = asyncio.get_running_loop()

    # Enqueue out of order.
    f2 = loop.create_future()
    controller.enqueue_item(
        batch_id="b1",
        expected_size=3,
        index=2,
        ctx=FakeCtx({"value": "c"}),
        future=f2,
        on_ready=lambda _batch_id: None,
    )

    f0 = loop.create_future()
    controller.enqueue_item(
        batch_id="b1",
        expected_size=3,
        index=0,
        ctx=FakeCtx({"value": "a"}),
        future=f0,
        on_ready=lambda _batch_id: None,
    )

    f1 = loop.create_future()
    controller.enqueue_item(
        batch_id="b1",
        expected_size=3,
        index=1,
        ctx=FakeCtx({"value": "b"}),
        future=f1,
        on_ready=lambda _batch_id: None,
    )

    controller.start_batch(
        batch_id="b1",
        expected_size=3,
        default_batch_size=3,
        on_ready=lambda _batch_id: None,
    )

    await runner._maybe_flush_batch(controller, "b1")

    assert await f0 == {"uppercase": "A"}
    assert await f1 == {"uppercase": "B"}
    assert await f2 == {"uppercase": "C"}


@pytest.mark.asyncio(loop_scope="session")
async def test_batch_flush_validates_output_length() -> None:
    class Inp(BaseModel):
        value: str

    hatchet = Hatchet(config=ClientConfig())
    wf = hatchet.workflow(name="batch-unit-test-length", input_validator=Inp)

    @wf.batch_task(name="step", batch_size=2)
    async def step(inputs: list[Inp], ctxs: list[Any]) -> list[dict[str, str]]:
        return [{"uppercase": inputs[0].value.upper()}]

    action_id = wf._create_action_name(step)
    runner = Runner(
        event_queue=queue.Queue(),
        config=hatchet.config,
        slots=10,
        handle_kill=False,
        action_registry={action_id: step},
        labels=None,
        lifespan_context=None,
        log_sender=object(),  # not used
    )
    controller = runner._batch_controllers[action_id]
    loop = asyncio.get_running_loop()

    futures = []
    for index, value in enumerate(["a", "b"]):
        fut = loop.create_future()
        futures.append(fut)
        controller.enqueue_item(
            batch_id="b1",
            expected_size=2,
            index=index,
            ctx=FakeCtx({"value": value}),
            future=fut,
            on_ready=lambda _batch_id: None,
        )

    controller.start_batch(
        batch_id="b1",
        expected_size=2,
        default_batch_size=2,
        on_ready=lambda _batch_id: None,
    )

    await runner._maybe_flush_batch(controller, "b1")
    results = await asyncio.gather(*futures, return_exceptions=True)
    assert all(isinstance(r, ValueError) for r in results)


@pytest.mark.asyncio(loop_scope="session")
async def test_batch_flush_fans_out_errors() -> None:
    class Inp(BaseModel):
        value: str

    hatchet = Hatchet(config=ClientConfig())
    wf = hatchet.workflow(name="batch-unit-test-errors", input_validator=Inp)

    @wf.batch_task(name="step", batch_size=2)
    async def step(inputs: list[Inp], ctxs: list[Any]) -> list[dict[str, str]]:
        raise RuntimeError("boom")

    action_id = wf._create_action_name(step)
    runner = Runner(
        event_queue=queue.Queue(),
        config=hatchet.config,
        slots=10,
        handle_kill=False,
        action_registry={action_id: step},
        labels=None,
        lifespan_context=None,
        log_sender=object(),  # not used
    )
    controller = runner._batch_controllers[action_id]
    loop = asyncio.get_running_loop()

    futures = []
    for index, value in enumerate(["a", "b"]):
        fut = loop.create_future()
        futures.append(fut)
        controller.enqueue_item(
            batch_id="b1",
            expected_size=2,
            index=index,
            ctx=FakeCtx({"value": value}),
            future=fut,
            on_ready=lambda _batch_id: None,
        )

    controller.start_batch(
        batch_id="b1",
        expected_size=2,
        default_batch_size=2,
        on_ready=lambda _batch_id: None,
    )

    await runner._maybe_flush_batch(controller, "b1")
    results = await asyncio.gather(*futures, return_exceptions=True)
    assert all(isinstance(r, RuntimeError) for r in results)

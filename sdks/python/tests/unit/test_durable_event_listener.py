from __future__ import annotations

import asyncio
from collections.abc import AsyncIterator
from typing import Any
from unittest.mock import AsyncMock, MagicMock, patch

import grpc
import grpc.aio
import pytest

from hatchet_sdk.clients.listeners.durable_event_listener import (
    DEFAULT_RECONNECT_INTERVAL,
    DurableEventListener,
    DurableTaskEventWaitForAck,
    WaitForEvent,
)
from hatchet_sdk.contracts.v1.dispatcher_pb2 import (
    DurableEventLogEntryRef,
    DurableTaskEventWaitForAckResponse,
    DurableTaskRequest,
    DurableTaskResponse,
)
from hatchet_sdk.contracts.v1.shared.condition_pb2 import DurableEventListenerConditions

_MODULE = "hatchet_sdk.clients.listeners.durable_event_listener"


class ControllableStream:
    def __init__(self) -> None:
        self._queue: asyncio.Queue[tuple[str, object]] = asyncio.Queue()

    def push(self, response: object) -> None:
        self._queue.put_nowait(("response", response))

    def end(self) -> None:
        self._queue.put_nowait(("end", None))

    def fail(self, error: BaseException) -> None:
        self._queue.put_nowait(("error", error))

    def __aiter__(self) -> ControllableStream:
        return self

    async def __anext__(self) -> object:
        kind, value = await self._queue.get()
        if kind == "end":
            raise StopAsyncIteration
        if kind == "error":
            raise value  # type: ignore[misc]
        return value


def _make_grpc_error(code: grpc.StatusCode, details: str = "") -> grpc.aio.AioRpcError:
    empty: grpc.aio.Metadata = grpc.aio.Metadata()
    return grpc.aio.AioRpcError(code, empty, empty, details)


class _Harness:
    def __init__(self) -> None:
        config = MagicMock()
        config.token = "test-token"
        admin_client = MagicMock()
        self.listener = DurableEventListener(config, admin_client)

        self.streams: list[ControllableStream] = []
        self.call_count = 0

        self._mock_conn = MagicMock()
        self._mock_conn.close = AsyncMock()

        self._patches: list[Any] = [
            patch(f"{_MODULE}.new_conn", return_value=self._mock_conn),
            patch(f"{_MODULE}.V1DispatcherStub", side_effect=self._make_stub),
            patch(f"{_MODULE}.create_authorization_header", return_value=[]),
            patch(f"{_MODULE}.DEFAULT_RECONNECT_INTERVAL", 0.01),
        ]
        for p in self._patches:
            p.start()

    def _make_stub(self, _channel: object) -> MagicMock:
        stub = MagicMock()
        stub.DurableTask.side_effect = self._next_stream
        return stub

    def _next_stream(self, *_a: object, **_kw: object) -> ControllableStream:
        idx = min(self.call_count, len(self.streams) - 1)
        self.call_count += 1
        return self.streams[idx]

    def add_eof_stream(self) -> ControllableStream:
        s = ControllableStream()
        s.end()
        self.streams.append(s)
        return s

    def add_hanging_stream(self) -> ControllableStream:
        s = ControllableStream()
        self.streams.append(s)
        return s

    def add_error_stream(self, error: BaseException) -> ControllableStream:
        s = ControllableStream()
        s.fail(error)
        self.streams.append(s)
        return s

    async def start(self, worker_id: str = "w1") -> None:
        await self.listener.start(worker_id)

    async def teardown(self) -> None:
        try:
            await self.listener.stop()
        except Exception:
            pass
        for s in self.streams:
            try:
                s.end()
            except Exception:
                pass
        for p in self._patches:
            p.stop()


@pytest.fixture
async def harness() -> AsyncIterator[_Harness]:
    h = _Harness()
    yield h
    await h.teardown()


async def test_opens_new_stream_after_eof(harness: _Harness) -> None:
    harness.add_eof_stream()
    harness.add_hanging_stream()

    await harness.start()
    await asyncio.sleep(0.15)

    assert harness.call_count >= 2


async def test_multiple_eof_reconnects(harness: _Harness) -> None:
    for _ in range(3):
        harness.add_eof_stream()
    harness.add_hanging_stream()

    await harness.start()
    await asyncio.sleep(0.3)

    assert harness.call_count >= 4


async def test_reconnects_on_unavailable(harness: _Harness) -> None:
    err = _make_grpc_error(grpc.StatusCode.UNAVAILABLE, "server unavailable")
    harness.add_error_stream(err)
    harness.add_hanging_stream()

    await harness.start()
    await asyncio.sleep(0.15)

    assert harness.call_count >= 2


async def test_reconnects_on_internal_error(harness: _Harness) -> None:
    err = _make_grpc_error(grpc.StatusCode.INTERNAL, "internal")
    harness.add_error_stream(err)
    harness.add_hanging_stream()

    await harness.start()
    await asyncio.sleep(0.15)

    assert harness.call_count >= 2


async def test_reconnects_on_generic_exception(harness: _Harness) -> None:
    s = ControllableStream()
    harness.streams.append(s)
    harness.add_hanging_stream()

    await harness.start()
    s.fail(RuntimeError("unexpected"))
    await asyncio.sleep(0.15)

    assert harness.call_count >= 2


async def test_breaks_out_on_grpc_cancelled(harness: _Harness) -> None:
    err = _make_grpc_error(grpc.StatusCode.CANCELLED, "cancelled")
    harness.add_error_stream(err)
    harness.add_hanging_stream()

    await harness.start()
    await asyncio.sleep(0.15)

    assert harness.call_count == 1


async def test_no_reconnect_after_stop(harness: _Harness) -> None:
    harness.add_hanging_stream()

    await harness.start()
    await harness.listener.stop()
    await asyncio.sleep(0.15)

    assert harness.call_count == 1


async def test_fail_pending_acks_clears_event_acks(harness: _Harness) -> None:
    harness.add_hanging_stream()
    await harness.start()

    future: asyncio.Future[object] = asyncio.get_event_loop().create_future()
    harness.listener._pending_event_acks[("task1", 1)] = future  # type: ignore[assignment]

    harness.listener._fail_pending_acks(ConnectionResetError("disconnected"))

    assert len(harness.listener._pending_event_acks) == 0
    with pytest.raises(ConnectionResetError, match="disconnected"):
        future.result()


async def test_pending_callbacks_survive_disconnect(harness: _Harness) -> None:
    harness.add_eof_stream()
    harness.add_hanging_stream()

    await harness.start()

    future: asyncio.Future[object] = asyncio.get_event_loop().create_future()
    future.add_done_callback(
        lambda f: f.exception() if f.done() and not f.cancelled() else None
    )
    harness.listener._pending_callbacks[("task1", 1, 0, 1)] = future  # type: ignore[assignment]

    await asyncio.sleep(0.15)

    assert not future.done()
    assert ("task1", 1, 0, 1) in harness.listener._pending_callbacks


async def test_fail_pending_acks_clears_eviction_acks_on_disconnect(
    harness: _Harness,
) -> None:
    harness.add_eof_stream()
    harness.add_hanging_stream()

    await harness.start()

    future: asyncio.Future[None] = asyncio.get_event_loop().create_future()
    harness.listener._pending_eviction_acks[("task1", 1)] = future

    await asyncio.sleep(0.15)

    assert future.done()


async def test_event_acks_rejected_when_stream_ends(harness: _Harness) -> None:
    stream1 = ControllableStream()
    harness.streams.append(stream1)
    harness.add_hanging_stream()

    await harness.start()
    await asyncio.sleep(0.05)

    future: asyncio.Future[object] = asyncio.get_event_loop().create_future()
    harness.listener._pending_event_acks[("task1", 1)] = future  # type: ignore[assignment]

    stream1.end()
    await asyncio.sleep(0.15)

    assert future.done()
    with pytest.raises(ConnectionResetError):
        future.result()


async def test_event_acks_rejected_when_stream_errors(harness: _Harness) -> None:
    stream1 = ControllableStream()
    harness.streams.append(stream1)
    harness.add_hanging_stream()

    await harness.start()
    await asyncio.sleep(0.05)

    future: asyncio.Future[object] = asyncio.get_event_loop().create_future()
    harness.listener._pending_event_acks[("task1", 1)] = future  # type: ignore[assignment]

    stream1.fail(_make_grpc_error(grpc.StatusCode.UNAVAILABLE, "gone"))
    await asyncio.sleep(0.15)

    assert future.done()
    with pytest.raises(ConnectionResetError):
        future.result()


async def test_request_queue_exists_after_each_connect(harness: _Harness) -> None:
    harness.add_eof_stream()
    harness.add_hanging_stream()

    await harness.start()
    await asyncio.sleep(0.15)

    assert harness.call_count >= 2
    assert harness.listener._request_queue is not None


async def test_survives_connect_failure_and_keeps_running(harness: _Harness) -> None:
    stream1 = ControllableStream()
    harness.streams.append(stream1)
    harness.add_hanging_stream()

    await harness.start()
    await asyncio.sleep(0.05)

    import hatchet_sdk.clients.listeners.durable_event_listener as mod

    original_new_conn = getattr(mod, "new_conn")
    setattr(mod, "new_conn", MagicMock(side_effect=ConnectionError("network down")))

    stream1.end()
    await asyncio.sleep(0.3)

    setattr(mod, "new_conn", original_new_conn)

    assert harness.listener._running is True


async def test_still_running_after_reconnect(harness: _Harness) -> None:
    harness.add_eof_stream()
    harness.add_hanging_stream()

    await harness.start()
    await asyncio.sleep(0.15)

    assert harness.listener._running is True


async def test_has_new_stream_after_reconnect(harness: _Harness) -> None:
    s1 = ControllableStream()
    harness.streams.append(s1)
    harness.add_hanging_stream()

    await harness.start()
    await asyncio.sleep(0.05)

    old_stream = harness.listener._stream
    s1.end()
    await asyncio.sleep(0.15)

    assert harness.listener._stream is not old_stream


def _wait_for_event() -> WaitForEvent:
    return WaitForEvent(
        wait_for_conditions=DurableEventListenerConditions(), label=None
    )


def _drain_queue(queue: asyncio.Queue[DurableTaskRequest]) -> list[DurableTaskRequest]:
    items = []
    while not queue.empty():
        items.append(queue.get_nowait())
    return items


async def test_send_event_during_reconnect_gap_survives(harness: _Harness) -> None:
    s1 = ControllableStream()
    harness.streams.append(s1)
    s2 = harness.add_hanging_stream()

    await harness.start()
    await asyncio.sleep(0.05)

    with patch(f"{_MODULE}.DEFAULT_RECONNECT_INTERVAL", 0.3):
        s1.end()
        await asyncio.sleep(0.05)

        send_task = asyncio.create_task(
            harness.listener.send_event("marker-task", 7, _wait_for_event())
        )
        await asyncio.sleep(0.05)
        assert not send_task.done()

        await asyncio.sleep(0.5)

    assert harness.call_count >= 2

    queue = harness.listener._request_queue
    assert queue is not None
    carried_over = [
        req
        for req in _drain_queue(queue)
        if req.HasField("wait_for")
        and req.wait_for.durable_task_external_id == "marker-task"
    ]
    assert carried_over, (
        "request enqueued during the reconnect gap was dropped — "
        "the durable run would hang forever awaiting its ack"
    )

    s2.push(
        DurableTaskResponse(
            wait_for_ack=DurableTaskEventWaitForAckResponse(
                ref=DurableEventLogEntryRef(
                    durable_task_external_id="marker-task",
                    invocation_count=7,
                    node_id=2,
                    branch_id=0,
                )
            )
        )
    )

    ack = await asyncio.wait_for(send_task, timeout=1.0)
    assert isinstance(ack, DurableTaskEventWaitForAck)
    assert ack.node_id == 2
    assert ack.invocation_count == 7


async def test_send_event_ack_timeout_raises(harness: _Harness) -> None:
    harness.add_hanging_stream()
    await harness.start()

    harness.listener._EVENT_ACK_TIMEOUT_S = 0.05

    with pytest.raises(TimeoutError, match="durable event ack"):
        await harness.listener.send_event("task-1", 1, _wait_for_event())

    assert ("task-1", 1) not in harness.listener._pending_event_acks


async def test_send_event_eviction_cancel_propagates_not_timeout(
    harness: _Harness,
) -> None:
    harness.add_hanging_stream()
    await harness.start()

    send_task = asyncio.create_task(
        harness.listener.send_event("task-1", 1, _wait_for_event())
    )
    await asyncio.sleep(0.05)

    harness.listener.cleanup_task_state("task-1", 1)

    with pytest.raises(asyncio.CancelledError):
        await send_task

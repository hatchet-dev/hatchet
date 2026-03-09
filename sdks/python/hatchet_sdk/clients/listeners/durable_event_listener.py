import asyncio
import json
from collections.abc import AsyncIterator, Callable
from contextlib import suppress
from dataclasses import dataclass
from datetime import timedelta
from typing import Annotated, Literal, cast

import grpc.aio
from pydantic import BaseModel, Field
from typing_extensions import Never, Self

from hatchet_sdk.clients.admin import (
    AdminClient,
    TriggerWorkflowOptions,
)
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.v1.dispatcher_pb2 import (
    DurableEventLogEntryRef,
    DurableTaskAwaitedCompletedEntry,
    DurableTaskCompleteMemoRequest,
    DurableTaskErrorType,
    DurableTaskEventLogEntryCompletedResponse,
    DurableTaskEvictInvocationRequest,
    DurableTaskMemoRequest,
    DurableTaskRequest,
    DurableTaskRequestRegisterWorker,
    DurableTaskResponse,
    DurableTaskTriggerRunsRequest,
    DurableTaskWaitForRequest,
    DurableTaskWorkerStatusRequest,
)
from hatchet_sdk.contracts.v1.dispatcher_pb2_grpc import V1DispatcherStub
from hatchet_sdk.contracts.v1.shared.condition_pb2 import DurableEventListenerConditions
from hatchet_sdk.exceptions import NonDeterminismError
from hatchet_sdk.logger import logger
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.utils.cache import TTLCache
from hatchet_sdk.utils.typing import JSONSerializableMapping

DEFAULT_RECONNECT_INTERVAL = 3  # seconds


@dataclass(frozen=True)
class WaitForEvent:
    wait_for_conditions: DurableEventListenerConditions


@dataclass(frozen=True)
class RunChildEvent:
    workflow_name: str
    input: str | None
    trigger_workflow_opts: TriggerWorkflowOptions


@dataclass(frozen=True)
class RunChildrenEvent:
    children: list[RunChildEvent]


@dataclass(frozen=True)
class MemoEvent:
    memo_key: bytes
    result: str | None


DurableTaskSendEvent = WaitForEvent | RunChildrenEvent | MemoEvent


class MaybeCachedMemoEntry(BaseModel):
    found: bool
    data: bytes | None = None


class DurableTaskRunAckEntry(BaseModel):
    node_id: int
    branch_id: int


class DurableTaskEventRunAck(BaseModel):
    ack_type: Literal["run"] = "run"
    invocation_count: int
    durable_task_external_id: str
    run_entries: list[DurableTaskRunAckEntry] = Field(default_factory=list)


class DurableTaskEventMemoAck(BaseModel):
    ack_type: Literal["memo"] = "memo"
    invocation_count: int
    durable_task_external_id: str
    branch_id: int
    node_id: int
    memo_already_existed: bool
    memo_result_payload: bytes | None = None


class DurableTaskEventWaitForAck(BaseModel):
    ack_type: Literal["wait"] = "wait"
    invocation_count: int
    durable_task_external_id: str
    branch_id: int
    node_id: int


DurableTaskEventAck = Annotated[
    DurableTaskEventRunAck | DurableTaskEventMemoAck | DurableTaskEventWaitForAck,
    Field(discriminator="ack_type"),
]


class DurableTaskEventLogEntryResult(BaseModel):
    durable_task_external_id: str
    node_id: int
    payload: JSONSerializableMapping | None

    @classmethod
    def from_proto(cls, proto: DurableTaskEventLogEntryCompletedResponse) -> Self:
        payload: JSONSerializableMapping | None = None
        if proto.payload:
            payload = json.loads(proto.payload.decode("utf-8"))

        return cls(
            durable_task_external_id=proto.ref.durable_task_external_id,
            node_id=proto.ref.node_id,
            payload=payload,
        )


TaskExternalId = str
NodeId = int
BranchId = int
InvocationCount = int

PendingCallback = tuple[TaskExternalId, InvocationCount, BranchId, NodeId]
PendingEventAck = tuple[TaskExternalId, InvocationCount]
PendingEvictionAck = tuple[TaskExternalId, InvocationCount]


class DurableEventListener:
    def __init__(
        self,
        config: ClientConfig,
        admin_client: AdminClient,
        on_server_evict: Callable[[str, int], None] | None = None,
    ):
        self.config = config
        self.token = config.token
        self.admin_client = admin_client

        self._worker_id: str | None = None

        self._conn: grpc.aio.Channel | None = None
        self._stub: V1DispatcherStub | None = None
        self._stream: (
            grpc.aio.StreamStreamCall[DurableTaskRequest, DurableTaskResponse] | None
        ) = None

        self._request_queue: asyncio.Queue[DurableTaskRequest] | None = None
        self._pending_event_acks: dict[
            PendingEventAck, asyncio.Future[DurableTaskEventAck]
        ] = {}
        self._pending_eviction_acks: dict[PendingEvictionAck, asyncio.Future[None]] = {}
        self._pending_callbacks: dict[
            PendingCallback, asyncio.Future[DurableTaskEventLogEntryResult]
        ] = {}

        # Completions that arrived before wait_for_callback() registered a
        # future in _pending_callbacks. This happens when the server delivers
        # an entry_completed between the event ack and the wait_for_callback
        # call (e.g. an already-satisfied sleep delivered via polling).
        self._buffered_completions: TTLCache[
            PendingCallback, DurableTaskEventLogEntryResult
        ] = TTLCache(ttl=timedelta(seconds=10))

        self._receive_task: asyncio.Task[None] | None = None
        self._send_task: asyncio.Task[None] | None = None
        self._running = False
        self._start_lock = asyncio.Lock()

        self._on_server_evict = on_server_evict

    @property
    def worker_id(self) -> str | None:
        return self._worker_id

    async def _connect(self) -> None:
        if self._conn is not None:
            with suppress(Exception):
                await self._conn.close()

        logger.info("durable event listener connecting...")

        self._conn = new_conn(self.config, aio=True)
        self._stub = V1DispatcherStub(self._conn)
        self._request_queue = asyncio.Queue()

        self._stream = cast(
            grpc.aio.StreamStreamCall[DurableTaskRequest, DurableTaskResponse],
            self._stub.DurableTask(
                self._request_iterator(),  # type: ignore[arg-type]
                metadata=get_metadata(self.token),
            ),
        )

        await self._register_worker()
        await self._poll_worker_status()
        logger.info("durable event listener connected")

    async def start(self, worker_id: str) -> None:
        async with self._start_lock:
            if self._running:
                return

            self._worker_id = worker_id
            self._running = True

            await self._connect()

            self._receive_task = asyncio.create_task(self._receive_loop())
            self._send_task = asyncio.create_task(self._send_loop())

    async def ensure_started(self, worker_id: str) -> None:
        if not self._running:
            await self.start(worker_id)

    async def stop(self) -> None:
        self._running = False
        self._buffered_completions.stop_eviction_job()

        if self._receive_task:
            self._receive_task.cancel()
            with suppress(asyncio.CancelledError):
                await self._receive_task

        if self._send_task:
            self._send_task.cancel()
            with suppress(asyncio.CancelledError):
                await self._send_task

        if self._conn:
            await self._conn.close()

    async def _request_iterator(self) -> AsyncIterator[DurableTaskRequest]:
        if not self._request_queue:
            raise RuntimeError("Request queue not initialized")

        while self._running:
            with suppress(asyncio.TimeoutError):
                yield await asyncio.wait_for(self._request_queue.get(), timeout=1.0)

    async def _send_loop(self) -> None:
        while self._running:
            await asyncio.sleep(1)
            await self._poll_worker_status()

    async def _poll_worker_status(self) -> None:
        if self._request_queue is None or self._worker_id is None:
            return

        if not self._pending_callbacks:
            return

        waiting = [
            DurableTaskAwaitedCompletedEntry(
                durable_task_external_id=task_ext_id,
                invocation_count=inv_count,
                node_id=node_id,
                branch_id=branch_id,
            )
            for (task_ext_id, inv_count, branch_id, node_id) in self._pending_callbacks
        ]

        request = DurableTaskRequest(
            worker_status=DurableTaskWorkerStatusRequest(
                worker_id=self._worker_id,
                waiting_entries=waiting,
            )
        )
        await self._request_queue.put(request)

    def _fail_pending_acks(self, exc: Exception) -> None:
        for future in self._pending_event_acks.values():
            if not future.done():
                future.set_exception(exc)
        self._pending_event_acks.clear()

        for eviction_future in self._pending_eviction_acks.values():
            if not eviction_future.done():
                eviction_future.set_exception(exc)
        self._pending_eviction_acks.clear()

    async def _receive_loop(self) -> None:
        while self._running:
            if not self._stream:
                await asyncio.sleep(DEFAULT_RECONNECT_INTERVAL)
                continue

            try:
                async for response in self._stream:
                    await self._handle_response(response)

                if self._running:
                    logger.warning(
                        f"durable event listener disconnected (EOF), reconnecting in {DEFAULT_RECONNECT_INTERVAL}s..."
                    )
                    self._fail_pending_acks(
                        ConnectionResetError("durable stream disconnected")
                    )
                    await asyncio.sleep(DEFAULT_RECONNECT_INTERVAL)
                    await self._connect()

            except grpc.aio.AioRpcError as e:
                if e.code() == grpc.StatusCode.CANCELLED:
                    break
                logger.warning(
                    f"durable event listener disconnected: code={e.code()}, details={e.details()}, reconnecting in {DEFAULT_RECONNECT_INTERVAL}s..."
                )
                if self._running:
                    self._fail_pending_acks(
                        ConnectionResetError(
                            f"durable stream error: {e.code()} {e.details()}"
                        )
                    )
                    await asyncio.sleep(DEFAULT_RECONNECT_INTERVAL)
                    try:
                        await self._connect()
                    except Exception:
                        logger.exception("failed to reconnect durable event listener")

            except asyncio.CancelledError:
                break

            except Exception as e:
                logger.exception(f"unexpected error in durable event listener: {e}")
                if self._running:
                    self._fail_pending_acks(e)
                    await asyncio.sleep(DEFAULT_RECONNECT_INTERVAL)
                    try:
                        await self._connect()
                    except Exception:
                        logger.exception("failed to reconnect durable event listener")

    async def _handle_response(self, response: DurableTaskResponse) -> None:
        if response.HasField("register_worker"):
            pass
        elif response.HasField("trigger_runs_ack"):
            trigger_ack = response.trigger_runs_ack
            event_key = (
                trigger_ack.durable_task_external_id,
                trigger_ack.invocation_count,
            )
            trigger_ack_future = self._pending_event_acks.pop(event_key, None)
            if trigger_ack_future is not None and not trigger_ack_future.done():
                trigger_ack_future.set_result(
                    DurableTaskEventRunAck(
                        invocation_count=trigger_ack.invocation_count,
                        durable_task_external_id=trigger_ack.durable_task_external_id,
                        run_entries=[
                            DurableTaskRunAckEntry(
                                node_id=e.node_id,
                                branch_id=e.branch_id,
                            )
                            for e in trigger_ack.run_entries
                        ],
                    )
                )
        elif response.HasField("memo_ack"):
            memo_ack = response.memo_ack
            event_key = (
                memo_ack.ref.durable_task_external_id,
                memo_ack.ref.invocation_count,
            )
            memo_ack_future = self._pending_event_acks.pop(event_key, None)
            if memo_ack_future is not None and not memo_ack_future.done():
                memo_ack_future.set_result(
                    DurableTaskEventMemoAck(
                        invocation_count=memo_ack.ref.invocation_count,
                        durable_task_external_id=memo_ack.ref.durable_task_external_id,
                        node_id=memo_ack.ref.node_id,
                        branch_id=memo_ack.ref.branch_id,
                        memo_already_existed=memo_ack.memo_already_existed,
                        memo_result_payload=memo_ack.memo_result_payload,
                    )
                )
        elif response.HasField("wait_for_ack"):
            wait_for_ack = response.wait_for_ack
            event_key = (
                wait_for_ack.ref.durable_task_external_id,
                wait_for_ack.ref.invocation_count,
            )
            wait_for_ack_future = self._pending_event_acks.pop(event_key, None)
            if wait_for_ack_future is not None and not wait_for_ack_future.done():
                wait_for_ack_future.set_result(
                    DurableTaskEventWaitForAck(
                        invocation_count=wait_for_ack.ref.invocation_count,
                        durable_task_external_id=wait_for_ack.ref.durable_task_external_id,
                        node_id=wait_for_ack.ref.node_id,
                        branch_id=wait_for_ack.ref.branch_id,
                    )
                )
        elif response.HasField("entry_completed"):
            completed = response.entry_completed
            completed_key = (
                completed.ref.durable_task_external_id,
                completed.ref.invocation_count,
                completed.ref.branch_id,
                completed.ref.node_id,
            )
            result = DurableTaskEventLogEntryResult.from_proto(completed)
            if completed_key in self._pending_callbacks:
                completed_future = self._pending_callbacks[completed_key]
                if not completed_future.done():
                    completed_future.set_result(result)
                del self._pending_callbacks[completed_key]
            else:
                self._buffered_completions[completed_key] = result
        elif response.HasField("eviction_ack"):
            eviction_ack = response.eviction_ack
            eviction_key = (
                eviction_ack.durable_task_external_id,
                eviction_ack.invocation_count,
            )
            if eviction_key in self._pending_eviction_acks:
                future = self._pending_eviction_acks.pop(eviction_key)
                if not future.done():
                    future.set_result(None)
        elif response.HasField("server_evict"):
            evict = response.server_evict
            logger.info(
                f"received server eviction notification for task {evict.durable_task_external_id} "
                f"invocation {evict.invocation_count}: {evict.reason}"
            )
            self.cleanup_task_state(
                evict.durable_task_external_id, evict.invocation_count
            )
            if self._on_server_evict is not None:
                self._on_server_evict(
                    evict.durable_task_external_id, evict.invocation_count
                )
        elif response.HasField("error"):
            error = response.error
            exc: Exception

            if (
                error.error_type
                == DurableTaskErrorType.DURABLE_TASK_ERROR_TYPE_NONDETERMINISM
            ):
                exc = NonDeterminismError(
                    task_external_id=error.ref.durable_task_external_id,
                    invocation_count=error.ref.invocation_count,
                    message=error.error_message,
                    node_id=error.ref.node_id,
                )
            else:
                ## fallthrough, this shouldn't happen unless we add an error type to the engine and the SDK
                ## hasn't been updated to handle it
                exc = Exception(
                    "Unspecified durable task error: "
                    + error.error_message
                    + f" (type: {error.error_type})"
                )

            event_key = (error.ref.durable_task_external_id, error.ref.invocation_count)
            if event_key in self._pending_event_acks:
                error_pending_ack_future = self._pending_event_acks.pop(event_key)
                if not error_pending_ack_future.done():
                    error_pending_ack_future.set_exception(exc)

            callback_key = (
                error.ref.durable_task_external_id,
                error.ref.invocation_count,
                error.ref.branch_id,
                error.ref.node_id,
            )

            if callback_key in self._pending_callbacks:
                error_pending_callback_future = self._pending_callbacks.pop(
                    callback_key
                )
                if not error_pending_callback_future.done():
                    error_pending_callback_future.set_exception(exc)

            error_eviction_key: PendingEvictionAck = (
                error.ref.durable_task_external_id,
                error.ref.invocation_count,
            )
            if error_eviction_key in self._pending_eviction_acks:
                eviction_future = self._pending_eviction_acks.pop(error_eviction_key)
                if not eviction_future.done():
                    eviction_future.set_exception(exc)

    async def _register_worker(self) -> None:
        if self._request_queue is None or self._worker_id is None:
            raise RuntimeError("Client not started")

        request = DurableTaskRequest(
            register_worker=DurableTaskRequestRegisterWorker(worker_id=self._worker_id)
        )
        await self._request_queue.put(request)

    async def send_event(
        self,
        durable_task_external_id: str,
        invocation_count: int,
        event: DurableTaskSendEvent,
    ) -> DurableTaskEventAck:
        if self._request_queue is None:
            raise RuntimeError("Client not started")

        key = (durable_task_external_id, invocation_count)
        future: asyncio.Future[DurableTaskEventAck] = asyncio.Future()
        self._pending_event_acks[key] = future

        request: DurableTaskRequest

        if isinstance(event, RunChildrenEvent):
            trigger_opts_list = [
                self.admin_client._create_workflow_run_request(
                    workflow_name=child.workflow_name,
                    input=child.input,
                    options=child.trigger_workflow_opts,
                )
                for child in event.children
            ]

            trigger_req = DurableTaskTriggerRunsRequest(
                durable_task_external_id=durable_task_external_id,
                invocation_count=invocation_count,
                trigger_opts=trigger_opts_list,
            )

            request = DurableTaskRequest(trigger_runs=trigger_req)

        elif isinstance(event, WaitForEvent):
            wait_req = DurableTaskWaitForRequest(
                durable_task_external_id=durable_task_external_id,
                invocation_count=invocation_count,
                wait_for_conditions=event.wait_for_conditions,
            )

            request = DurableTaskRequest(wait_for=wait_req)
        elif isinstance(event, MemoEvent):
            memo_req = DurableTaskMemoRequest(
                durable_task_external_id=durable_task_external_id,
                invocation_count=invocation_count,
                key=event.memo_key,
            )

            if event.result is not None:
                memo_req.payload = event.result.encode("utf-8")

            request = DurableTaskRequest(memo=memo_req)

        else:
            e: Never = event
            raise ValueError(f"Unknown durable task send event: {e}")

        await self._request_queue.put(request)

        return await future

    async def wait_for_callback(
        self,
        durable_task_external_id: str,
        invocation_count: int,
        branch_id: int,
        node_id: int,
    ) -> DurableTaskEventLogEntryResult:
        key = (durable_task_external_id, invocation_count, branch_id, node_id)

        if key in self._buffered_completions:
            return self._buffered_completions.pop(key)

        if key not in self._pending_callbacks:
            future: asyncio.Future[DurableTaskEventLogEntryResult] = asyncio.Future()
            self._pending_callbacks[key] = future
            await self._poll_worker_status()

        return await self._pending_callbacks[key]

    def cleanup_task_state(
        self, durable_task_external_id: str, invocation_count: int
    ) -> None:
        """Remove pending callbacks, acks, and buffered completions for old invocations of a task."""
        stale_cb_keys = [
            k
            for k in self._pending_callbacks
            if k[0] == durable_task_external_id and k[1] <= invocation_count
        ]
        for k in stale_cb_keys:
            fut = self._pending_callbacks.pop(k)
            if not fut.done():
                fut.cancel()

        stale_ack_keys = [
            ak
            for ak in self._pending_event_acks
            if ak[0] == durable_task_external_id and ak[1] <= invocation_count
        ]
        for ak in stale_ack_keys:
            ack_fut = self._pending_event_acks.pop(ak)
            if not ack_fut.done():
                ack_fut.cancel()

        stale_early_keys = [
            ek
            for ek in self._buffered_completions
            if ek[0] == durable_task_external_id and ek[1] <= invocation_count
        ]
        for ek in stale_early_keys:
            del self._buffered_completions[ek]

    _EVICTION_ACK_TIMEOUT_S = 30.0

    async def send_evict_invocation(
        self,
        durable_task_external_id: str,
        invocation_count: int,
        reason: str | None = None,
    ) -> None:
        """Send an eviction request to the server and wait for acknowledgement."""
        if self._request_queue is None:
            raise RuntimeError("Client not started")

        eviction_key: PendingEvictionAck = (
            durable_task_external_id,
            invocation_count,
        )
        ack_future: asyncio.Future[None] = asyncio.Future()
        self._pending_eviction_acks[eviction_key] = ack_future

        req = DurableTaskEvictInvocationRequest(
            durable_task_external_id=durable_task_external_id,
            invocation_count=invocation_count,
        )
        if reason is not None:
            req.reason = reason

        request = DurableTaskRequest(evict_invocation=req)
        await self._request_queue.put(request)

        try:
            await asyncio.wait_for(ack_future, timeout=self._EVICTION_ACK_TIMEOUT_S)
        except asyncio.TimeoutError as err:
            self._pending_eviction_acks.pop(eviction_key, None)
            raise TimeoutError(
                f"Eviction ack timed out after {self._EVICTION_ACK_TIMEOUT_S:.0f}s "
                f"for task {durable_task_external_id} invocation {invocation_count}"
            ) from err

    async def send_memo_completed_notification(
        self,
        durable_task_external_id: str,
        node_id: int,
        branch_id: int,
        invocation_count: int,
        memo_key: bytes,
        memo_result_payload: bytes | None,
    ) -> None:
        if self._request_queue is None:
            raise RuntimeError("Client not started")

        await self._request_queue.put(
            DurableTaskRequest(
                complete_memo=DurableTaskCompleteMemoRequest(
                    ref=DurableEventLogEntryRef(
                        durable_task_external_id=durable_task_external_id,
                        node_id=node_id,
                        invocation_count=invocation_count,
                        branch_id=branch_id,
                    ),
                    memo_key=memo_key,
                    payload=memo_result_payload,
                )
            )
        )

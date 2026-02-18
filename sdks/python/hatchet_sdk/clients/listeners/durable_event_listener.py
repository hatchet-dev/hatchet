import asyncio
import json
from collections.abc import AsyncIterator
from contextlib import suppress
from typing import Self, cast

import grpc.aio
from pydantic import BaseModel

from hatchet_sdk.clients.admin import AdminClient, TriggerWorkflowOptions
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.v1.dispatcher_pb2 import (
    DurableTaskAwaitedCompletedEntry,
    DurableTaskErrorType,
    DurableTaskEventKind,
    DurableTaskEventLogEntryCompletedResponse,
    DurableTaskEventRequest,
    DurableTaskRequest,
    DurableTaskRequestRegisterWorker,
    DurableTaskResponse,
    DurableTaskWorkerStatusRequest,
)
from hatchet_sdk.contracts.v1.dispatcher_pb2_grpc import V1DispatcherStub
from hatchet_sdk.contracts.v1.shared.condition_pb2 import DurableEventListenerConditions
from hatchet_sdk.exceptions import NonDeterminismError
from hatchet_sdk.logger import logger
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.utils.typing import JSONSerializableMapping

DEFAULT_RECONNECT_INTERVAL = 3  # seconds


class DurableTaskEventAck(BaseModel):
    invocation_count: int
    durable_task_external_id: str
    node_id: int


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
            durable_task_external_id=proto.durable_task_external_id,
            node_id=proto.node_id,
            payload=payload,
        )


class DurableEventListener:
    def __init__(self, config: ClientConfig, admin_client: AdminClient):
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
            tuple[str, int], asyncio.Future[DurableTaskEventAck]
        ] = {}
        self._pending_callbacks: dict[
            tuple[str, int], asyncio.Future[DurableTaskEventLogEntryResult]
        ] = {}

        self._receive_task: asyncio.Task[None] | None = None
        self._send_task: asyncio.Task[None] | None = None
        self._running = False
        self._start_lock = asyncio.Lock()

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
                node_id=node_id,
            )
            for (task_ext_id, node_id) in self._pending_callbacks
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
        if response.HasField("reset"):
            print("Received reset response for durable task ")
            # pass
        elif response.HasField("trigger_ack"):
            trigger_ack = response.trigger_ack
            event_key = (
                trigger_ack.durable_task_external_id,
                trigger_ack.invocation_count,
            )
            if event_key in self._pending_event_acks:
                self._pending_event_acks[event_key].set_result(
                    DurableTaskEventAck(
                        invocation_count=trigger_ack.invocation_count,
                        durable_task_external_id=trigger_ack.durable_task_external_id,
                        node_id=trigger_ack.node_id,
                    )
                )
                del self._pending_event_acks[event_key]
        elif response.HasField("entry_completed"):
            completed = response.entry_completed
            completed_key = (
                completed.durable_task_external_id,
                completed.node_id,
            )
            if completed_key in self._pending_callbacks:
                completed_future = self._pending_callbacks[completed_key]
                if not completed_future.done():
                    completed_future.set_result(
                        DurableTaskEventLogEntryResult.from_proto(completed)
                    )
                del self._pending_callbacks[completed_key]
        elif response.HasField("error"):
            error = response.error
            exc: Exception

            if (
                error.error_type
                == DurableTaskErrorType.DURABLE_TASK_ERROR_TYPE_NONDETERMINISM
            ):
                exc = NonDeterminismError(
                    task_external_id=error.durable_task_external_id,
                    invocation_count=error.invocation_count,
                    message=error.error_message,
                    node_id=error.node_id,
                )
            else:
                ## fallthrough, this shouldn't happen unless we add an error type to the engine and the SDK
                ## hasn't been updated to handle it
                exc = Exception(
                    "Unspecified durable task error: "
                    + error.error_message
                    + f" (type: {error.error_type})"
                )

            event_key = (error.durable_task_external_id, error.invocation_count)
            if event_key in self._pending_event_acks:
                error_pending_ack_future = self._pending_event_acks.pop(event_key)
                if not error_pending_ack_future.done():
                    error_pending_ack_future.set_exception(exc)

            callback_key = (error.durable_task_external_id, error.node_id)
            if callback_key in self._pending_callbacks:
                error_pending_callback_future = self._pending_callbacks.pop(
                    callback_key
                )
                if not error_pending_callback_future.done():
                    error_pending_callback_future.set_exception(exc)

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
        kind: DurableTaskEventKind,
        payload: JSONSerializableMapping | None = None,
        wait_for_conditions: DurableEventListenerConditions | None = None,
        # todo: combine these? or separate methods? or overload?
        workflow_name: str | None = None,
        trigger_workflow_opts: TriggerWorkflowOptions | None = None,
    ) -> DurableTaskEventAck:
        if self._request_queue is None:
            raise RuntimeError("Client not started")

        key = (durable_task_external_id, invocation_count)
        future: asyncio.Future[DurableTaskEventAck] = asyncio.Future()
        self._pending_event_acks[key] = future

        _trigger_opts = (
            self.admin_client._create_workflow_run_request(
                workflow_name=workflow_name,
                input=payload or {},
                options=trigger_workflow_opts or TriggerWorkflowOptions(),
            )
            if workflow_name
            else None
        )

        event_request = DurableTaskEventRequest(
            durable_task_external_id=durable_task_external_id,
            invocation_count=invocation_count,
            kind=kind,
            trigger_opts=_trigger_opts,
        )

        if payload is not None:
            event_request.payload = json.dumps(payload).encode("utf-8")

        if wait_for_conditions is not None:
            event_request.wait_for_conditions.CopyFrom(wait_for_conditions)

        request = DurableTaskRequest(event=event_request)
        await self._request_queue.put(request)

        return await future

    async def wait_for_callback(
        self,
        durable_task_external_id: str,
        node_id: int,
    ) -> DurableTaskEventLogEntryResult:
        key = (durable_task_external_id, node_id)

        if key not in self._pending_callbacks:
            future: asyncio.Future[DurableTaskEventLogEntryResult] = asyncio.Future()
            self._pending_callbacks[key] = future

        return await self._pending_callbacks[key]

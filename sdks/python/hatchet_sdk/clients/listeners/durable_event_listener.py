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
from hatchet_sdk.logger import logger
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.utils.typing import JSONSerializableMapping


class DurableTaskEventAck(BaseModel):
    invocation_count: int
    durable_task_external_id: str
    node_id: int


class DurableTaskCallbackResult(BaseModel):
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
            tuple[str, int], asyncio.Future[DurableTaskCallbackResult]
        ] = {}

        self._receive_task: asyncio.Task[None] | None = None
        self._send_task: asyncio.Task[None] | None = None
        self._running = False
        self._start_lock = asyncio.Lock()

    @property
    def worker_id(self) -> str | None:
        return self._worker_id

    async def start(self, worker_id: str) -> None:
        async with self._start_lock:
            if self._running:
                return

            self._worker_id = worker_id
            self._running = True
            self._request_queue = asyncio.Queue()

            self._conn = new_conn(self.config, aio=True)
            self._stub = V1DispatcherStub(self._conn)

            self._stream = cast(
                grpc.aio.StreamStreamCall[DurableTaskRequest, DurableTaskResponse],
                self._stub.DurableTask(
                    self._request_iterator(),  # type: ignore[arg-type]
                    metadata=get_metadata(self.token),
                ),
            )

            self._receive_task = asyncio.create_task(self._receive_loop())
            self._send_task = asyncio.create_task(self._send_loop())

            await self._register_worker()

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

    async def _receive_loop(self) -> None:
        if not self._stream:
            return

        try:
            async for response in self._stream:
                await self._handle_response(response)
        except grpc.aio.AioRpcError as e:
            if e.code() != grpc.StatusCode.CANCELLED:
                logger.error(
                    f"DurableTask stream error: code={e.code()}, details={e.details()}"
                )
        except asyncio.CancelledError:
            logger.debug("Receive loop cancelled")
        except Exception as e:
            logger.exception(f"Unexpected error in receive loop: {e}")

    async def _handle_response(self, response: DurableTaskResponse) -> None:
        if response.HasField("register_worker"):
            logger.info(
                f"Registered durable task worker: {response.register_worker.worker_id}"
            )
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
                future = self._pending_callbacks[completed_key]
                if not future.done():
                    future.set_result(DurableTaskCallbackResult.from_proto(completed))
                del self._pending_callbacks[completed_key]

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
    ) -> DurableTaskCallbackResult:
        key = (durable_task_external_id, node_id)

        if key not in self._pending_callbacks:
            future: asyncio.Future[DurableTaskCallbackResult] = asyncio.Future()
            self._pending_callbacks[key] = future

        return await self._pending_callbacks[key]

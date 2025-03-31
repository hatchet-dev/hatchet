import asyncio
import json
from collections.abc import AsyncIterator
from typing import Any, cast

import grpc
import grpc.aio
from grpc._cython import cygrpc  # type: ignore[attr-defined]

from hatchet_sdk.clients.event_ts import ThreadSafeEvent, read_with_interrupt
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.dispatcher_pb2 import (
    SubscribeToWorkflowRunsRequest,
    WorkflowRunEvent,
)
from hatchet_sdk.contracts.dispatcher_pb2_grpc import DispatcherStub
from hatchet_sdk.logger import logger
from hatchet_sdk.metadata import get_metadata

DEFAULT_WORKFLOW_LISTENER_RETRY_INTERVAL = 3  # seconds
DEFAULT_WORKFLOW_LISTENER_RETRY_COUNT = 5
DEFAULT_WORKFLOW_LISTENER_INTERRUPT_INTERVAL = 1800  # 30 minutes

DEDUPE_MESSAGE = "DUPLICATE_WORKFLOW_RUN"


class _Subscription:
    def __init__(self, id: int, workflow_run_id: str):
        self.id = id
        self.workflow_run_id = workflow_run_id
        self.queue: asyncio.Queue[WorkflowRunEvent | None] = asyncio.Queue()

    async def __aiter__(self) -> "_Subscription":
        return self

    async def __anext__(self) -> WorkflowRunEvent | None:
        return await self.queue.get()

    async def get(self) -> WorkflowRunEvent:
        event = await self.queue.get()

        if event is None:
            raise StopAsyncIteration

        return event

    async def put(self, item: WorkflowRunEvent) -> None:
        await self.queue.put(item)

    async def close(self) -> None:
        await self.queue.put(None)


class PooledWorkflowRunListener:
    def __init__(self, config: ClientConfig):
        self.token = config.token
        self.config = config

        # list of all active subscriptions, mapping from a subscription id to a workflow run id
        self.subscriptions_to_workflows: dict[int, str] = {}

        # list of workflow run ids mapped to an array of subscription ids
        self.workflows_to_subscriptions: dict[str, list[int]] = {}

        self.subscription_counter: int = 0
        self.subscription_counter_lock: asyncio.Lock = asyncio.Lock()

        self.requests: asyncio.Queue[SubscribeToWorkflowRunsRequest | int] = (
            asyncio.Queue()
        )

        self.listener: (
            grpc.aio.UnaryStreamCall[SubscribeToWorkflowRunsRequest, WorkflowRunEvent]
            | None
        ) = None
        self.listener_task: asyncio.Task[None] | None = None

        self.curr_requester: int = 0

        # events have keys of the format workflow_run_id + subscription_id
        self.events: dict[int, _Subscription] = {}

        self.interrupter: asyncio.Task[None] | None = None

        ## IMPORTANT: This needs to be created lazily so we don't require
        ## an event loop to instantiate the client.
        self.client: DispatcherStub | None = None

    async def _interrupter(self) -> None:
        """
        _interrupter runs in a separate thread and interrupts the listener according to a configurable duration.
        """
        await asyncio.sleep(DEFAULT_WORKFLOW_LISTENER_INTERRUPT_INTERVAL)

        if self.interrupt is not None:
            self.interrupt.set()

    async def _init_producer(self) -> None:
        try:
            if not self.listener:
                while True:
                    try:
                        self.listener = await self._retry_subscribe()

                        logger.debug("Workflow run listener connected.")

                        # spawn an interrupter task
                        if self.interrupter is not None and not self.interrupter.done():
                            self.interrupter.cancel()

                        self.interrupter = asyncio.create_task(self._interrupter())

                        while True:
                            self.interrupt = ThreadSafeEvent()
                            if self.listener is None:
                                continue

                            t = asyncio.create_task(
                                read_with_interrupt(self.listener, self.interrupt)
                            )
                            await self.interrupt.wait()

                            if not t.done():
                                # print a warning
                                logger.warning(
                                    "Interrupted read_with_interrupt task of workflow run listener"
                                )

                                t.cancel()
                                self.listener.cancel()

                                await asyncio.sleep(
                                    DEFAULT_WORKFLOW_LISTENER_RETRY_INTERVAL
                                )
                                break

                            workflow_event: WorkflowRunEvent = t.result()

                            if workflow_event is cygrpc.EOF:
                                break

                            # get a list of subscriptions for this workflow
                            subscriptions = self.workflows_to_subscriptions.get(
                                workflow_event.workflowRunId, []
                            )

                            for subscription_id in subscriptions:
                                await self.events[subscription_id].put(workflow_event)

                    except grpc.RpcError as e:
                        logger.debug(f"grpc error in workflow run listener: {e}")
                        await asyncio.sleep(DEFAULT_WORKFLOW_LISTENER_RETRY_INTERVAL)
                        continue

        except Exception as e:
            logger.error(f"Error in workflow run listener: {e}")

            self.listener = None

            # close all subscriptions
            for subscription_id in self.events:
                await self.events[subscription_id].close()

            raise e

    async def _request(self) -> AsyncIterator[SubscribeToWorkflowRunsRequest]:
        self.curr_requester = self.curr_requester + 1

        # replay all existing subscriptions
        workflow_run_set = set(self.subscriptions_to_workflows.values())

        for workflow_run_id in workflow_run_set:
            yield SubscribeToWorkflowRunsRequest(
                workflowRunId=workflow_run_id,
            )

        while True:
            request = await self.requests.get()

            # if the request is an int which matches the current requester, then we should stop
            if request == self.curr_requester:
                break

            # if we've gotten an int that doesn't match the current requester, then we should ignore it
            if isinstance(request, int):
                continue

            yield request
            self.requests.task_done()

    def cleanup_subscription(self, subscription_id: int) -> None:
        workflow_run_id = self.subscriptions_to_workflows[subscription_id]

        if workflow_run_id in self.workflows_to_subscriptions:
            self.workflows_to_subscriptions[workflow_run_id].remove(subscription_id)

        del self.subscriptions_to_workflows[subscription_id]
        del self.events[subscription_id]

    async def subscribe(self, workflow_run_id: str) -> WorkflowRunEvent:
        subscription_id: int | None = None

        try:
            # create a new subscription id, place a mutex on the counter
            await self.subscription_counter_lock.acquire()
            self.subscription_counter += 1
            subscription_id = self.subscription_counter
            self.subscription_counter_lock.release()

            self.subscriptions_to_workflows[subscription_id] = workflow_run_id

            if workflow_run_id not in self.workflows_to_subscriptions:
                self.workflows_to_subscriptions[workflow_run_id] = [subscription_id]
            else:
                self.workflows_to_subscriptions[workflow_run_id].append(subscription_id)

            self.events[subscription_id] = _Subscription(
                subscription_id, workflow_run_id
            )

            await self.requests.put(
                SubscribeToWorkflowRunsRequest(
                    workflowRunId=workflow_run_id,
                )
            )

            if not self.listener_task or self.listener_task.done():
                self.listener_task = asyncio.create_task(self._init_producer())

            return await self.events[subscription_id].get()
        except asyncio.CancelledError:
            raise
        finally:
            if subscription_id:
                self.cleanup_subscription(subscription_id)

    async def aio_result(self, workflow_run_id: str) -> dict[str, Any]:
        from hatchet_sdk.clients.admin import DedupeViolationErr

        event = await self.subscribe(workflow_run_id)
        errors = [result.error for result in event.results if result.error]

        if errors:
            if DEDUPE_MESSAGE in errors[0]:
                raise DedupeViolationErr(errors[0])
            else:
                raise Exception(f"Workflow Errors: {errors}")

        return {
            result.stepReadableId: json.loads(result.output)
            for result in event.results
            if result.output
        }

    async def _retry_subscribe(
        self,
    ) -> grpc.aio.UnaryStreamCall[SubscribeToWorkflowRunsRequest, WorkflowRunEvent]:
        retries = 0
        if self.client is None:
            conn = new_conn(self.config, True)
            self.client = DispatcherStub(conn)

        while retries < DEFAULT_WORKFLOW_LISTENER_RETRY_COUNT:
            try:
                if retries > 0:
                    await asyncio.sleep(DEFAULT_WORKFLOW_LISTENER_RETRY_INTERVAL)

                # signal previous async iterator to stop
                if self.curr_requester != 0:
                    self.requests.put_nowait(self.curr_requester)

                return cast(
                    grpc.aio.UnaryStreamCall[
                        SubscribeToWorkflowRunsRequest, WorkflowRunEvent
                    ],
                    self.client.SubscribeToWorkflowRuns(
                        self._request(),  # type: ignore[arg-type]
                        metadata=get_metadata(self.token),
                    ),
                )
            except grpc.RpcError as e:
                if e.code() == grpc.StatusCode.UNAVAILABLE:
                    retries = retries + 1
                else:
                    raise ValueError(f"gRPC error: {e}")

        raise ValueError("Failed to connect to workflow run listener")

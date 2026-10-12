from __future__ import annotations

from collections.abc import Awaitable
from typing import cast

import pytest
import tenacity

from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.contracts.dispatcher_pb2 import (
    STEP_EVENT_TYPE_COMPLETED,
    STEP_EVENT_TYPE_FAILED,
    ActionEventResponse,
)
from hatchet_sdk.contracts.dispatcher_pb2_grpc import DispatcherStub
from hatchet_sdk.runnables.action import Action, BatchEventItem
from hatchet_sdk.runnables.types import BatchMemberId

MAX_ATTEMPTS = 20


class _AckCall:
    """Generated grpc.aio unary call that returns queued acks.

    grpc.aio returns `None` when it cannot deserialize the response.
    """

    def __init__(self, acks: list[ActionEventResponse | None]) -> None:
        self.acks = acks
        self.calls = 0

    def __call__(
        self, request: object, *, metadata: tuple[tuple[str, str]]
    ) -> Awaitable[ActionEventResponse | None]:
        ack = self.acks[min(self.calls, len(self.acks) - 1)]
        self.calls += 1

        async def response() -> ActionEventResponse | None:
            return ack

        return response()


class _FakeDispatcherStub:
    def __init__(self, call: _AckCall) -> None:
        self.SendStepActionEvent = call
        self.SendBatchActionEvent = call


@pytest.fixture(autouse=True)
def no_retry_wait(monkeypatch: pytest.MonkeyPatch) -> None:
    for method in (
        DispatcherClient._try_send_step_action_event,
        DispatcherClient._try_aio_send_batch_action_event,
    ):
        monkeypatch.setattr(method.retry, "wait", tenacity.wait_none())


def _client(call: _AckCall) -> DispatcherClient:
    client = DispatcherClient.__new__(DispatcherClient)
    client.config = ClientConfig.model_construct(token="token")
    client.token = "token"
    client.aio_channel = None
    client.client = None
    client.aio_client = cast(DispatcherStub, _FakeDispatcherStub(call))

    return client


def _action() -> Action:
    return Action.model_construct(
        worker_id="worker",
        job_id="job",
        job_run_id="job-run",
        step_id="step",
        step_run_id="step-run",
        action_id="action",
        retry_count=0,
        batch_id="batch",
    )


async def test_step_action_event_retries_missing_ack() -> None:
    ack = ActionEventResponse(tenant_id="tenant", worker_id="worker")
    call = _AckCall([None, ack])

    result = await _client(call).send_step_action_event(
        _action(), STEP_EVENT_TYPE_COMPLETED, "{}", should_not_retry=False
    )

    assert result == ack
    assert call.calls == 2


async def test_batch_action_event_retries_missing_ack() -> None:
    ack = ActionEventResponse(tenant_id="tenant", worker_id="worker")
    call = _AckCall([None, ack])

    result = await _client(call).aio_send_batch_action_event(
        _action(),
        STEP_EVENT_TYPE_FAILED,
        [BatchEventItem(task_run_external_id=BatchMemberId("task-run"))],
    )

    assert result == ack
    assert call.calls == 2


async def test_empty_ack_is_accepted_without_retry() -> None:
    call = _AckCall([ActionEventResponse()])

    result = await _client(call).send_step_action_event(
        _action(), STEP_EVENT_TYPE_COMPLETED, "{}", should_not_retry=False
    )

    assert result == ActionEventResponse()
    assert call.calls == 1


async def test_missing_ack_on_every_attempt_is_logged_and_returns_none(
    caplog: pytest.LogCaptureFixture,
) -> None:
    call = _AckCall([None])

    result = await _client(call).send_step_action_event(
        _action(), STEP_EVENT_TYPE_COMPLETED, "{}", should_not_retry=False
    )

    assert result is None
    assert call.calls == MAX_ATTEMPTS
    assert "failed to send step action event" in caplog.text

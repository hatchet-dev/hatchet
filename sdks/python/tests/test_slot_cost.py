"""Unit tests for the public slot_cost task parameter."""

import base64
import json
from typing import Any

import pytest

from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet


@pytest.fixture(scope="session", autouse=True)
def worker() -> Any:
    # These tests run offline, so override conftest's engine-backed worker fixture.
    yield None


def _offline_token() -> str:
    # A well-formed JWT so ClientConfig can construct offline. The client reads its claims and does
    # not check the signature.
    def segment(data: dict[str, str]) -> str:
        return base64.urlsafe_b64encode(json.dumps(data).encode()).rstrip(b"=").decode()

    header = segment({"alg": "none", "typ": "JWT"})
    payload = segment(
        {
            "sub": "tenant-offline",
            "server_url": "https://localhost",
            "grpc_broadcast_address": "localhost:7070",
        }
    )
    return f"{header}.{payload}.signature"


def dummy(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"foo": "bar"}


async def dummy_durable(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:
    return {"foo": "bar"}


@pytest.fixture
def hatchet(monkeypatch: pytest.MonkeyPatch) -> Hatchet:
    monkeypatch.setenv("HATCHET_CLIENT_TOKEN", _offline_token())
    return Hatchet()


def test_slot_cost_maps_to_default_pool(hatchet: Hatchet) -> None:
    wf = hatchet.workflow(name="slot-cost-wf")
    t = wf.task(slot_cost=5)(dummy)

    assert t.to_proto("svc").slot_requests == {"default": 5}


def test_omitting_slot_cost_keeps_one_default_slot(hatchet: Hatchet) -> None:
    wf = hatchet.workflow(name="slot-cost-wf")
    t = wf.task()(dummy)

    assert t.to_proto("svc").slot_requests == {"default": 1}


def test_slot_cost_one_is_accepted(hatchet: Hatchet) -> None:
    wf = hatchet.workflow(name="slot-cost-wf")
    t = wf.task(slot_cost=1)(dummy)

    assert t.to_proto("svc").slot_requests == {"default": 1}


@pytest.mark.parametrize("bad", [0, -1, -5])
def test_non_positive_slot_cost_is_rejected(hatchet: Hatchet, bad: int) -> None:
    wf = hatchet.workflow(name="slot-cost-wf")

    with pytest.raises(ValueError):
        wf.task(slot_cost=bad)(dummy)


def test_standalone_task_slot_cost(hatchet: Hatchet) -> None:
    standalone = hatchet.task(name="standalone-heavy", slot_cost=5)(dummy)

    assert standalone._task.to_proto("svc").slot_requests == {"default": 5}


def test_durable_task_is_unchanged(hatchet: Hatchet) -> None:
    wf = hatchet.workflow(name="slot-cost-wf")
    t = wf.durable_task()(dummy_durable)

    assert t.to_proto("svc").slot_requests == {"durable": 1}

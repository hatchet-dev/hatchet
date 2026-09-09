"""Unit tests for duplicate action-id handling in Worker.register_workflow.

Regression tests for https://github.com/hatchet-dev/hatchet/issues/4922 : action ids are
namespaced and lowercased, so tasks whose names differ only by case (or share an explicit
name) used to silently overwrite each other in the worker's action registry, leaving one of
the workflows routing to the wrong function with no warning.
"""

import base64
import json
from typing import Any
from unittest.mock import MagicMock

import pytest

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.worker.worker import Worker


def _fake_token() -> str:
    def b64(data: bytes) -> str:
        return base64.urlsafe_b64encode(data).rstrip(b"=").decode()

    claims = {
        "sub": "00000000-0000-0000-0000-000000000000",
        "server_url": "http://localhost:8080",
        "grpc_broadcast_address": "localhost:7077",
    }
    return ".".join([b64(json.dumps({"alg": "none"}).encode()), b64(json.dumps(claims).encode()), b64(b"sig")])


def _make_worker(hatchet: Hatchet) -> Worker:
    worker = hatchet.worker("registry-test-worker", slots=1, workflows=[])
    # Stub the only network call register_workflow makes.
    worker._client.admin.put_workflow = MagicMock()
    return worker


def test_case_only_name_difference_raises() -> None:
    hatchet = Hatchet(config=ClientConfig(token=_fake_token()))

    @hatchet.task(name="SendEmail")
    def send_email_a(input: EmptyModel, ctx: Context) -> dict[str, Any]:
        return {"who": "a"}

    @hatchet.task(name="sendemail")
    def send_email_b(input: EmptyModel, ctx: Context) -> dict[str, Any]:
        return {"who": "b"}

    worker = _make_worker(hatchet)
    worker.register_workflow(send_email_a)

    with pytest.raises(ValueError, match="already registered"):
        worker.register_workflow(send_email_b)

    # The first registration must be intact - no silent overwrite happened.
    assert list(worker._action_registry.keys()) == ["sendemail:sendemail"]


def test_same_explicit_name_on_two_functions_raises() -> None:
    hatchet = Hatchet(config=ClientConfig(token=_fake_token()))

    @hatchet.task(name="process")
    def process_email(input: EmptyModel, ctx: Context) -> dict[str, Any]:
        return {"who": "email"}

    @hatchet.task(name="process")
    def process_billing(input: EmptyModel, ctx: Context) -> dict[str, Any]:
        return {"who": "billing"}

    worker = _make_worker(hatchet)
    worker.register_workflow(process_email)

    with pytest.raises(ValueError, match="already registered"):
        worker.register_workflow(process_billing)


def test_distinct_names_still_register() -> None:
    hatchet = Hatchet(config=ClientConfig(token=_fake_token()))

    @hatchet.task(name="charge")
    def charge(input: EmptyModel, ctx: Context) -> dict[str, Any]:
        return {}

    @hatchet.task(name="notify")
    def notify(input: EmptyModel, ctx: Context) -> dict[str, Any]:
        return {}

    worker = _make_worker(hatchet)
    worker.register_workflow(charge)
    worker.register_workflow(notify)

    assert sorted(worker._action_registry.keys()) == ["charge:charge", "notify:notify"]


def test_reregistering_same_workflow_is_idempotent() -> None:
    hatchet = Hatchet(config=ClientConfig(token=_fake_token()))

    @hatchet.task(name="sendemail")
    def send_email(input: EmptyModel, ctx: Context) -> dict[str, Any]:
        return {}

    worker = _make_worker(hatchet)
    worker.register_workflow(send_email)
    worker.register_workflow(send_email)

    assert list(worker._action_registry.keys()) == ["sendemail:sendemail"]

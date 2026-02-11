import asyncio
from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager
from datetime import datetime, timezone
from typing import Any
from uuid import uuid4

import aiohttp
import pytest

from examples.webhook_with_scope.worker import (
    WebhookInputWithScope,
    WebhookInputWithStaticPayload,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_event import V1Event
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary
from hatchet_sdk.clients.rest.models.v1_webhook import V1Webhook
from hatchet_sdk.clients.rest.models.v1_webhook_basic_auth import V1WebhookBasicAuth
from hatchet_sdk.clients.rest.models.v1_webhook_source_name import V1WebhookSourceName
from hatchet_sdk.features.webhooks import CreateWebhookRequest

TEST_BASIC_USERNAME = "test_user"
TEST_BASIC_PASSWORD = "test_password"


@pytest.fixture
def webhook_body() -> WebhookInputWithScope:
    return WebhookInputWithScope(type="test", message="Hello, world!")


@pytest.fixture
def webhook_body_with_scope() -> WebhookInputWithScope:
    return WebhookInputWithScope(
        type="test", message="Hello, world!", scope="test-scope-value"
    )


@pytest.fixture
def webhook_body_for_static() -> WebhookInputWithStaticPayload:
    return WebhookInputWithStaticPayload(type="test", message="Hello, world!")


@pytest.fixture
def test_run_id() -> str:
    return str(uuid4())


@pytest.fixture
def test_start() -> datetime:
    return datetime.now(timezone.utc)


async def send_webhook_request(
    url: str,
    body: dict[str, Any],
    username: str = TEST_BASIC_USERNAME,
    password: str = TEST_BASIC_PASSWORD,
) -> aiohttp.ClientResponse:
    auth = aiohttp.BasicAuth(username, password)

    async with aiohttp.ClientSession() as session:
        return await session.post(url, json=body, auth=auth)


async def wait_for_event(
    hatchet: Hatchet,
    webhook_name: str,
    test_start: datetime,
) -> V1Event | None:
    await asyncio.sleep(5)

    events = await hatchet.event.aio_list(since=test_start)

    if events.rows is None:
        return None

    return next(
        (
            event
            for event in events.rows
            if event.triggering_webhook_name == webhook_name
        ),
        None,
    )


async def wait_for_workflow_run(
    hatchet: Hatchet, event_id: str, test_start: datetime
) -> V1TaskSummary | None:
    await asyncio.sleep(5)

    runs = await hatchet.runs.aio_list(
        since=test_start,
        additional_metadata={
            "hatchet__event_id": event_id,
        },
    )

    if len(runs.rows) == 0:
        return None

    return runs.rows[0]


@asynccontextmanager
async def webhook_with_scope_expression(
    hatchet: Hatchet,
    test_run_id: str,
    scope_expression: str,
    event_key_expression: str | None = None,
    username: str = TEST_BASIC_USERNAME,
    password: str = TEST_BASIC_PASSWORD,
) -> AsyncGenerator[V1Webhook, None]:

    if event_key_expression is None:
        event_key_expression = (
            f"'{hatchet.config.apply_namespace('webhook-scope')}:' + input.type"
        )

    webhook_request = CreateWebhookRequest(
        source_name=V1WebhookSourceName.GENERIC,
        name=f"test-webhook-scope-{test_run_id}",
        event_key_expression=event_key_expression,
        scope_expression=scope_expression,
        auth_type="BASIC",
        auth=V1WebhookBasicAuth(
            username=username,
            password=password,
        ),
    )

    incoming_webhook = hatchet.webhooks.create(webhook_request)

    try:
        yield incoming_webhook
    finally:
        hatchet.webhooks.delete(incoming_webhook.name)


@asynccontextmanager
async def webhook_with_static_payload(
    hatchet: Hatchet,
    test_run_id: str,
    static_payload: dict[str, Any],
    event_key_expression: str | None = None,
    username: str = TEST_BASIC_USERNAME,
    password: str = TEST_BASIC_PASSWORD,
) -> AsyncGenerator[V1Webhook, None]:

    if event_key_expression is None:
        event_key_expression = (
            f"'{hatchet.config.apply_namespace('webhook-static')}:' + input.type"
        )

    webhook_request = CreateWebhookRequest(
        source_name=V1WebhookSourceName.GENERIC,
        name=f"test-webhook-static-{test_run_id}",
        event_key_expression=event_key_expression,
        static_payload=static_payload,
        auth_type="BASIC",
        auth=V1WebhookBasicAuth(
            username=username,
            password=password,
        ),
    )

    incoming_webhook = hatchet.webhooks.create(webhook_request)

    try:
        yield incoming_webhook
    finally:
        hatchet.webhooks.delete(incoming_webhook.name)


@asynccontextmanager
async def webhook_with_scope_and_static(
    hatchet: Hatchet,
    test_run_id: str,
    scope_expression: str,
    static_payload: dict[str, Any],
    event_key_expression: str | None = None,
    username: str = TEST_BASIC_USERNAME,
    password: str = TEST_BASIC_PASSWORD,
) -> AsyncGenerator[V1Webhook, None]:

    if event_key_expression is None:
        event_key_expression = (
            f"'{hatchet.config.apply_namespace('webhook-scope')}:' + input.type"
        )

    webhook_request = CreateWebhookRequest(
        source_name=V1WebhookSourceName.GENERIC,
        name=f"test-webhook-both-{test_run_id}",
        event_key_expression=event_key_expression,
        scope_expression=scope_expression,
        static_payload=static_payload,
        auth_type="BASIC",
        auth=V1WebhookBasicAuth(
            username=username,
            password=password,
        ),
    )

    incoming_webhook = hatchet.webhooks.create(webhook_request)

    try:
        yield incoming_webhook
    finally:
        hatchet.webhooks.delete(incoming_webhook.name)


def url(tenant_id: str, webhook_name: str) -> str:
    return f"http://localhost:8080/api/v1/stable/tenants/{tenant_id}/webhooks/{webhook_name}"


async def assert_has_runs(
    hatchet: Hatchet,
    test_start: datetime,
    expected_event_key: str,
    expected_payload: dict[str, Any],
    incoming_webhook: V1Webhook,
) -> None:
    triggered_event = await wait_for_event(hatchet, incoming_webhook.name, test_start)
    assert triggered_event is not None
    assert triggered_event.key == expected_event_key
    assert triggered_event.payload == expected_payload

    workflow_run = await wait_for_workflow_run(
        hatchet, triggered_event.metadata.id, test_start
    )
    assert workflow_run is not None
    assert workflow_run.status == V1TaskStatus.COMPLETED
    assert workflow_run.additional_metadata is not None

    assert (
        workflow_run.additional_metadata["hatchet__event_id"]
        == triggered_event.metadata.id
    )
    assert workflow_run.additional_metadata["hatchet__event_key"] == triggered_event.key


async def assert_event_created_with_payload(
    hatchet: Hatchet,
    test_start: datetime,
    expected_payload: dict[str, Any],
    incoming_webhook: V1Webhook,
) -> None:
    triggered_event = await wait_for_event(hatchet, incoming_webhook.name, test_start)
    assert triggered_event is not None
    assert triggered_event.payload == expected_payload


@pytest.mark.asyncio(loop_scope="session")
async def test_scope_expression_from_payload(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body_with_scope: WebhookInputWithScope,
) -> None:
    async with webhook_with_scope_expression(
        hatchet,
        test_run_id,
        scope_expression="input.scope",
    ) as incoming_webhook:
        assert incoming_webhook.scope_expression == "input.scope"

        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            webhook_body_with_scope.model_dump(),
        ) as response:
            assert response.status == 200
            data = await response.json()
            assert data == {"message": "ok"}

        triggered_event = await wait_for_event(
            hatchet, incoming_webhook.name, test_start
        )
        assert triggered_event is not None
        assert triggered_event.scope == webhook_body_with_scope.scope


@pytest.mark.asyncio(loop_scope="session")
async def test_scope_expression_from_headers(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body: WebhookInputWithScope,
) -> None:
    async with webhook_with_scope_expression(
        hatchet,
        test_run_id,
        scope_expression="headers['x-custom-scope']",
    ) as incoming_webhook:
        assert incoming_webhook.scope_expression == "headers['x-custom-scope']"

        auth = aiohttp.BasicAuth(TEST_BASIC_USERNAME, TEST_BASIC_PASSWORD)
        async with aiohttp.ClientSession() as session:
            async with await session.post(
                url(hatchet.tenant_id, incoming_webhook.name),
                json=webhook_body.model_dump(),
                auth=auth,
                headers={"X-Custom-Scope": "header-scope-value"},
            ) as response:
                assert response.status == 200
                data = await response.json()
                assert data == {"message": "ok"}

        triggered_event = await wait_for_event(
            hatchet, incoming_webhook.name, test_start
        )
        assert triggered_event is not None
        assert triggered_event.scope == "header-scope-value"


@pytest.mark.asyncio(loop_scope="session")
async def test_scope_expression_concatenation(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body_with_scope: WebhookInputWithScope,
) -> None:
    async with webhook_with_scope_expression(
        hatchet,
        test_run_id,
        scope_expression="'prefix:' + input.type + ':' + input.scope",
    ) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            webhook_body_with_scope.model_dump(),
        ) as response:
            assert response.status == 200

        triggered_event = await wait_for_event(
            hatchet, incoming_webhook.name, test_start
        )
        assert triggered_event is not None
        assert (
            triggered_event.scope
            == f"prefix:{webhook_body_with_scope.type}:{webhook_body_with_scope.scope}"
        )


@pytest.mark.asyncio(loop_scope="session")
async def test_static_payload_adds_fields(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body_for_static: WebhookInputWithStaticPayload,
) -> None:
    static_payload = {
        "source": "webhook-test",
        "environment": "test",
    }

    async with webhook_with_static_payload(
        hatchet,
        test_run_id,
        static_payload=static_payload,
    ) as incoming_webhook:
        assert incoming_webhook.static_payload == static_payload

        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            webhook_body_for_static.model_dump(),
        ) as response:
            assert response.status == 200
            data = await response.json()
            assert data == {"message": "ok"}

        expected_payload = {
            **webhook_body_for_static.model_dump(),
            **static_payload,
        }

        await assert_event_created_with_payload(
            hatchet, test_start, expected_payload, incoming_webhook
        )


@pytest.mark.asyncio(loop_scope="session")
async def test_static_payload_overrides_existing_fields(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
) -> None:
    incoming_body = {
        "type": "test",
        "message": "Hello, world!",
        "source": "original-source",
    }

    static_payload = {
        "source": "static-source",
        "environment": "production",
    }

    async with webhook_with_static_payload(
        hatchet,
        test_run_id,
        static_payload=static_payload,
    ) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            incoming_body,
        ) as response:
            assert response.status == 200

        expected_payload = {
            "type": "test",
            "message": "Hello, world!",
            "source": "static-source",
            "environment": "production",
        }

        await assert_event_created_with_payload(
            hatchet, test_start, expected_payload, incoming_webhook
        )


@pytest.mark.asyncio(loop_scope="session")
async def test_scope_expression_uses_static_payload_values(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
) -> None:
    incoming_body = {
        "type": "test",
        "message": "Hello, world!",
    }

    static_payload = {
        "customer_id": "cust-123",
        "environment": "production",
    }

    async with webhook_with_scope_and_static(
        hatchet,
        test_run_id,
        scope_expression="input.customer_id",
        static_payload=static_payload,
    ) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            incoming_body,
        ) as response:
            assert response.status == 200

        triggered_event = await wait_for_event(
            hatchet, incoming_webhook.name, test_start
        )
        assert triggered_event is not None

        assert triggered_event.scope == "cust-123"

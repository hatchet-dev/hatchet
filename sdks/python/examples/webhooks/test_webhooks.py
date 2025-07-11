import asyncio
from contextlib import asynccontextmanager
from datetime import datetime, timedelta, timezone
from typing import AsyncGenerator
from uuid import uuid4

import aiohttp
import pytest

from examples.webhooks.worker import WebhookInput
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.api.webhook_api import WebhookApi
from hatchet_sdk.clients.rest.models.v1_create_webhook_request import (
    V1CreateWebhookRequest,
)
from hatchet_sdk.clients.rest.models.v1_create_webhook_request_basic_auth import (
    V1CreateWebhookRequestBasicAuth,
)
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_webhook import V1Webhook
from hatchet_sdk.clients.rest.models.v1_webhook_auth_type import V1WebhookAuthType
from hatchet_sdk.clients.rest.models.v1_webhook_basic_auth import V1WebhookBasicAuth
from hatchet_sdk.clients.rest.models.v1_webhook_source_name import V1WebhookSourceName


@asynccontextmanager
async def webhook(
    hatchet: Hatchet,
    test_run_id: str,
) -> AsyncGenerator[V1Webhook, None]:
    client = hatchet.metrics.client()
    webhook_api = WebhookApi(client)

    basic_auth = V1CreateWebhookRequestBasicAuth(
        sourceName=V1WebhookSourceName.GENERIC,
        name=f"test-webhook-{test_run_id}",
        eventKeyExpression="'webhook:' + input.type",
        authType="BASIC",
        auth=V1WebhookBasicAuth(
            username="test_user",
            password="test_password",
        ),
    )

    incoming_webhook = webhook_api.v1_webhook_create(
        tenant=hatchet.tenant_id,
        v1_create_webhook_request=V1CreateWebhookRequest(basic_auth),
    )

    try:
        yield incoming_webhook
    finally:
        webhook_api.v1_webhook_delete(
            tenant=hatchet.tenant_id,
            v1_webhook=incoming_webhook.name,
        )


@pytest.mark.asyncio(loop_scope="session")
async def test_webhook_worklfow(hatchet: Hatchet) -> None:
    test_start = datetime.now(timezone.utc)
    test_run_id = str(uuid4())

    async with webhook(hatchet, test_run_id) as incoming_webhook:
        url = f"http://localhost:8080/api/v1/stable/tenants/{hatchet.tenant_id}/webhooks/{incoming_webhook.name}"
        body = WebhookInput(type="test", message="Hello, world!")

        async with aiohttp.ClientSession() as session:
            async with session.post(
                url,
                json=body.model_dump(),
                auth=aiohttp.BasicAuth("test_user", "test_password"),
            ) as response:
                assert response.status == 200
                data = await response.json()

                assert data == {"message": "ok"}

        await asyncio.sleep(1)

        events = await hatchet.event.aio_list(
            since=test_start,
        )

        assert events.rows is not None
        assert len(events.rows) > 0

        triggered_event = next(
            (
                event
                for event in events.rows
                if event.triggering_webhook_name == incoming_webhook.name
            ),
            None,
        )

        assert triggered_event is not None
        assert triggered_event.key == f"webhook:{body.type}"
        assert triggered_event.payload == body.model_dump()

        await asyncio.sleep(5)

        runs = await hatchet.runs.aio_list(
            since=test_start,
            additional_metadata={
                "hatchet__event_id": triggered_event.metadata.id,
            },
        )

        assert runs.rows is not None
        assert len(runs.rows) == 1

        run = runs.rows[0]

        assert run.additional_metadata is not None

        assert (
            run.additional_metadata["hatchet__event_id"] == triggered_event.metadata.id
        )
        assert run.additional_metadata["hatchet__event_key"] == triggered_event.key
        assert run.status == V1TaskStatus.COMPLETED


@pytest.mark.parametrize(
    "username,password",
    [
        ("test_user", "incorrect_password"),
        ("incorrect_user", "test_password"),
        ("incorrect_user", "incorrect_password"),
        ("", ""),
    ],
)
@pytest.mark.asyncio(loop_scope="session")
async def test_webhook_validation_failure(
    hatchet: Hatchet, username: str, password: str
) -> None:
    test_start = datetime.now(timezone.utc)
    test_run_id = str(uuid4())

    async with webhook(hatchet, test_run_id) as incoming_webhook:
        url = f"http://localhost:8080/api/v1/stable/tenants/{hatchet.tenant_id}/webhooks/{incoming_webhook.name}"
        body = WebhookInput(type="test", message="Hello, world!")

        async with aiohttp.ClientSession() as session:
            async with session.post(
                url,
                json=body.model_dump(),
                auth=aiohttp.BasicAuth(username, password),
            ) as response:
                assert response.status == 403

        await asyncio.sleep(1)

        events = await hatchet.event.aio_list(
            since=test_start,
        )

        assert events.rows is not None

        triggered_event = next(
            (
                event
                for event in events.rows
                if event.triggering_webhook_name == incoming_webhook.name
            ),
            None,
        )

        assert triggered_event is None

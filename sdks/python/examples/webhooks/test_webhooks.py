import asyncio
import base64
import hashlib
import hmac
import json
from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager
from datetime import datetime, timezone
from typing import Any
from uuid import uuid4

import aiohttp
import pytest

from examples.webhooks.worker import WebhookInput
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_event import V1Event
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary
from hatchet_sdk.clients.rest.models.v1_webhook import V1Webhook
from hatchet_sdk.clients.rest.models.v1_webhook_api_key_auth import V1WebhookAPIKeyAuth
from hatchet_sdk.clients.rest.models.v1_webhook_basic_auth import V1WebhookBasicAuth
from hatchet_sdk.clients.rest.models.v1_webhook_hmac_algorithm import (
    V1WebhookHMACAlgorithm,
)
from hatchet_sdk.clients.rest.models.v1_webhook_hmac_auth import V1WebhookHMACAuth
from hatchet_sdk.clients.rest.models.v1_webhook_hmac_encoding import (
    V1WebhookHMACEncoding,
)
from hatchet_sdk.clients.rest.models.v1_webhook_source_name import V1WebhookSourceName

TEST_BASIC_USERNAME = "test_user"
TEST_BASIC_PASSWORD = "test_password"
TEST_API_KEY_HEADER = "X-API-Key"
TEST_API_KEY_VALUE = "test_api_key_123"
TEST_HMAC_SIGNATURE_HEADER = "X-Signature"
TEST_HMAC_SECRET = "test_hmac_secret"


hatchet = Hatchet(debug=True)


@pytest.fixture
def webhook_body() -> WebhookInput:
    return WebhookInput(type="test", message="Hello, world!")


@pytest.fixture
def test_run_id() -> str:
    return str(uuid4())


@pytest.fixture
def test_start() -> datetime:
    return datetime.now(timezone.utc)


def create_hmac_signature(
    payload: bytes,
    secret: str,
    algorithm: V1WebhookHMACAlgorithm = V1WebhookHMACAlgorithm.SHA256,
    encoding: V1WebhookHMACEncoding = V1WebhookHMACEncoding.HEX,
) -> str:
    algorithm_map = {
        V1WebhookHMACAlgorithm.SHA1: hashlib.sha1,
        V1WebhookHMACAlgorithm.SHA256: hashlib.sha256,
        V1WebhookHMACAlgorithm.SHA512: hashlib.sha512,
        V1WebhookHMACAlgorithm.MD5: hashlib.md5,
    }

    hash_func = algorithm_map[algorithm]
    signature = hmac.new(secret.encode(), payload, hash_func).digest()

    if encoding == V1WebhookHMACEncoding.HEX:
        return signature.hex()
    if encoding == V1WebhookHMACEncoding.BASE64:
        return base64.b64encode(signature).decode()
    if encoding == V1WebhookHMACEncoding.BASE64URL:
        return base64.urlsafe_b64encode(signature).decode()

    raise ValueError(f"Unsupported encoding: {encoding}")


async def send_webhook_request(
    url: str,
    body: WebhookInput,
    auth_type: str,
    auth_data: dict[str, Any] | None = None,
    headers: dict[str, str] | None = None,
) -> aiohttp.ClientResponse:
    request_headers = headers or {}
    auth = None

    if auth_type == "BASIC" and auth_data:
        auth = aiohttp.BasicAuth(auth_data["username"], auth_data["password"])
    elif auth_type == "API_KEY" and auth_data:
        request_headers[auth_data["header_name"]] = auth_data["api_key"]
    elif auth_type == "HMAC" and auth_data:
        payload = json.dumps(body.model_dump()).encode()
        signature = create_hmac_signature(
            payload,
            auth_data["secret"],
            auth_data.get("algorithm", V1WebhookHMACAlgorithm.SHA256),
            auth_data.get("encoding", V1WebhookHMACEncoding.HEX),
        )
        request_headers[auth_data["header_name"]] = signature

    async with aiohttp.ClientSession() as session:
        return await session.post(
            url, json=body.model_dump(), auth=auth, headers=request_headers
        )


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
async def basic_auth_webhook(
    hatchet: Hatchet,
    test_run_id: str,
    username: str = TEST_BASIC_USERNAME,
    password: str = TEST_BASIC_PASSWORD,
    source_name: V1WebhookSourceName = V1WebhookSourceName.GENERIC,
) -> AsyncGenerator[V1Webhook, None]:

    incoming_webhook = hatchet.webhooks.create(
        source_name=source_name,
        name=f"test-webhook-basic-{test_run_id}",
        event_key_expression=f"'{hatchet.config.apply_namespace('webhook')}:' + input.type",
        auth=V1WebhookBasicAuth(username=username, password=password),
    )

    try:
        yield incoming_webhook
    finally:
        hatchet.webhooks.delete(incoming_webhook.name)


@asynccontextmanager
async def api_key_webhook(
    hatchet: Hatchet,
    test_run_id: str,
    header_name: str = TEST_API_KEY_HEADER,
    api_key: str = TEST_API_KEY_VALUE,
    source_name: V1WebhookSourceName = V1WebhookSourceName.GENERIC,
) -> AsyncGenerator[V1Webhook, None]:

    incoming_webhook = hatchet.webhooks.create(
        source_name=source_name,
        name=f"test-webhook-apikey-{test_run_id}",
        event_key_expression=f"'{hatchet.config.apply_namespace('webhook')}:' + input.type",
        auth=V1WebhookAPIKeyAuth(
            headerName=header_name,
            apiKey=api_key,
        ),
    )

    try:
        yield incoming_webhook
    finally:
        hatchet.webhooks.delete(incoming_webhook.name)


@asynccontextmanager
async def hmac_webhook(
    hatchet: Hatchet,
    test_run_id: str,
    signature_header_name: str = TEST_HMAC_SIGNATURE_HEADER,
    signing_secret: str = TEST_HMAC_SECRET,
    algorithm: V1WebhookHMACAlgorithm = V1WebhookHMACAlgorithm.SHA256,
    encoding: V1WebhookHMACEncoding = V1WebhookHMACEncoding.HEX,
    source_name: V1WebhookSourceName = V1WebhookSourceName.GENERIC,
) -> AsyncGenerator[V1Webhook, None]:

    incoming_webhook = hatchet.webhooks.create(
        source_name=source_name,
        name=f"test-webhook-hmac-{test_run_id}",
        event_key_expression=f"'{hatchet.config.apply_namespace('webhook')}:' + input.type",
        auth=V1WebhookHMACAuth(
            algorithm=algorithm,
            encoding=encoding,
            signatureHeaderName=signature_header_name,
            signingSecret=signing_secret,
        ),
    )

    try:
        yield incoming_webhook
    finally:
        hatchet.webhooks.delete(incoming_webhook.name)


def url(tenant_id: str, webhook_name: str) -> str:
    return f"http://localhost:8080/api/v1/stable/tenants/{tenant_id}/webhooks/{webhook_name}"


async def assert_has_runs(
    hatchet: Hatchet,
    test_start: datetime,
    webhook_body: WebhookInput,
    incoming_webhook: V1Webhook,
) -> None:
    triggered_event = await wait_for_event(hatchet, incoming_webhook.name, test_start)
    assert triggered_event is not None
    assert (
        triggered_event.key
        == f"{hatchet.config.apply_namespace('webhook')}:{webhook_body.type}"
    )
    assert triggered_event.payload == webhook_body.model_dump()

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
    assert workflow_run.status == V1TaskStatus.COMPLETED


async def assert_event_not_created(
    hatchet: Hatchet,
    test_start: datetime,
    incoming_webhook: V1Webhook,
) -> None:
    triggered_event = await wait_for_event(hatchet, incoming_webhook.name, test_start)
    assert triggered_event is None


@pytest.mark.asyncio(loop_scope="session")
async def test_basic_auth_success(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body: WebhookInput,
) -> None:
    async with basic_auth_webhook(hatchet, test_run_id) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            webhook_body,
            "BASIC",
            {"username": TEST_BASIC_USERNAME, "password": TEST_BASIC_PASSWORD},
        ) as response:
            assert response.status == 200
            data = await response.json()
            assert data == {"message": "ok"}

        await assert_has_runs(
            hatchet,
            test_start,
            webhook_body,
            incoming_webhook,
        )


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
async def test_basic_auth_failure(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body: WebhookInput,
    username: str,
    password: str,
) -> None:
    """Test basic authentication failures."""
    async with basic_auth_webhook(hatchet, test_run_id) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            webhook_body,
            "BASIC",
            {"username": username, "password": password},
        ) as response:
            assert response.status == 403

        await assert_event_not_created(
            hatchet,
            test_start,
            incoming_webhook,
        )


@pytest.mark.asyncio(loop_scope="session")
async def test_basic_auth_missing_credentials(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body: WebhookInput,
) -> None:
    async with basic_auth_webhook(hatchet, test_run_id) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name), webhook_body, "NONE"
        ) as response:
            assert response.status == 403

        await assert_event_not_created(
            hatchet,
            test_start,
            incoming_webhook,
        )


@pytest.mark.asyncio(loop_scope="session")
async def test_api_key_success(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body: WebhookInput,
) -> None:
    async with api_key_webhook(hatchet, test_run_id) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            webhook_body,
            "API_KEY",
            {"header_name": TEST_API_KEY_HEADER, "api_key": TEST_API_KEY_VALUE},
        ) as response:
            assert response.status == 200
            data = await response.json()
            assert data == {"message": "ok"}

        await assert_has_runs(
            hatchet,
            test_start,
            webhook_body,
            incoming_webhook,
        )


@pytest.mark.parametrize(
    "api_key",
    [
        "incorrect_api_key",
        "",
        "partial_key",
    ],
)
@pytest.mark.asyncio(loop_scope="session")
async def test_api_key_failure(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body: WebhookInput,
    api_key: str,
) -> None:
    async with api_key_webhook(hatchet, test_run_id) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            webhook_body,
            "API_KEY",
            {"header_name": TEST_API_KEY_HEADER, "api_key": api_key},
        ) as response:
            assert response.status == 403

        await assert_event_not_created(
            hatchet,
            test_start,
            incoming_webhook,
        )


@pytest.mark.asyncio(loop_scope="session")
async def test_api_key_missing_header(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body: WebhookInput,
) -> None:
    async with api_key_webhook(hatchet, test_run_id) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name), webhook_body, "NONE"
        ) as response:
            assert response.status == 403

        await assert_event_not_created(
            hatchet,
            test_start,
            incoming_webhook,
        )


@pytest.mark.asyncio(loop_scope="session")
async def test_hmac_success(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body: WebhookInput,
) -> None:
    async with hmac_webhook(hatchet, test_run_id) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            webhook_body,
            "HMAC",
            {
                "header_name": TEST_HMAC_SIGNATURE_HEADER,
                "secret": TEST_HMAC_SECRET,
                "algorithm": V1WebhookHMACAlgorithm.SHA256,
                "encoding": V1WebhookHMACEncoding.HEX,
            },
        ) as response:
            assert response.status == 200
            data = await response.json()
            assert data == {"message": "ok"}

        await assert_has_runs(
            hatchet,
            test_start,
            webhook_body,
            incoming_webhook,
        )


@pytest.mark.parametrize(
    "algorithm,encoding",
    [
        (V1WebhookHMACAlgorithm.SHA1, V1WebhookHMACEncoding.HEX),
        (V1WebhookHMACAlgorithm.SHA256, V1WebhookHMACEncoding.BASE64),
        (V1WebhookHMACAlgorithm.SHA512, V1WebhookHMACEncoding.BASE64URL),
        (V1WebhookHMACAlgorithm.MD5, V1WebhookHMACEncoding.HEX),
    ],
)
@pytest.mark.asyncio(loop_scope="session")
async def test_hmac_different_algorithms_and_encodings(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body: WebhookInput,
    algorithm: V1WebhookHMACAlgorithm,
    encoding: V1WebhookHMACEncoding,
) -> None:
    async with hmac_webhook(
        hatchet, test_run_id, algorithm=algorithm, encoding=encoding
    ) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            webhook_body,
            "HMAC",
            {
                "header_name": TEST_HMAC_SIGNATURE_HEADER,
                "secret": TEST_HMAC_SECRET,
                "algorithm": algorithm,
                "encoding": encoding,
            },
        ) as response:
            assert response.status == 200
            data = await response.json()
            assert data == {"message": "ok"}

        await assert_has_runs(
            hatchet,
            test_start,
            webhook_body,
            incoming_webhook,
        )


@pytest.mark.parametrize(
    "secret",
    [
        "incorrect_secret",
        "",
        "partial_secret",
    ],
)
@pytest.mark.asyncio(loop_scope="session")
async def test_hmac_signature_failure(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body: WebhookInput,
    secret: str,
) -> None:
    async with hmac_webhook(hatchet, test_run_id) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            webhook_body,
            "HMAC",
            {
                "header_name": TEST_HMAC_SIGNATURE_HEADER,
                "secret": secret,
                "algorithm": V1WebhookHMACAlgorithm.SHA256,
                "encoding": V1WebhookHMACEncoding.HEX,
            },
        ) as response:
            assert response.status == 403

        await assert_event_not_created(
            hatchet,
            test_start,
            incoming_webhook,
        )


@pytest.mark.asyncio(loop_scope="session")
async def test_hmac_missing_signature_header(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body: WebhookInput,
) -> None:
    async with hmac_webhook(hatchet, test_run_id) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name), webhook_body, "NONE"
        ) as response:
            assert response.status == 403

        await assert_event_not_created(
            hatchet,
            test_start,
            incoming_webhook,
        )


@pytest.mark.parametrize(
    "source_name",
    [
        V1WebhookSourceName.GENERIC,
        V1WebhookSourceName.GITHUB,
    ],
)
@pytest.mark.asyncio(loop_scope="session")
async def test_different_source_types(
    hatchet: Hatchet,
    test_run_id: str,
    test_start: datetime,
    webhook_body: WebhookInput,
    source_name: V1WebhookSourceName,
) -> None:
    async with basic_auth_webhook(
        hatchet, test_run_id, source_name=source_name
    ) as incoming_webhook:
        async with await send_webhook_request(
            url(hatchet.tenant_id, incoming_webhook.name),
            webhook_body,
            "BASIC",
            {"username": TEST_BASIC_USERNAME, "password": TEST_BASIC_PASSWORD},
        ) as response:
            assert response.status == 200
            data = await response.json()
            assert data == {"message": "ok"}

        await assert_has_runs(
            hatchet,
            test_start,
            webhook_body,
            incoming_webhook,
        )

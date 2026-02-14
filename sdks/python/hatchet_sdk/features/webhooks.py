from __future__ import annotations

import asyncio
from typing import Any

from hatchet_sdk.clients.rest.api.webhook_api import WebhookApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.v1_create_webhook_request import (
    V1CreateWebhookRequest,
)
from hatchet_sdk.clients.rest.models.v1_create_webhook_request_api_key import (
    V1CreateWebhookRequestAPIKey,
)
from hatchet_sdk.clients.rest.models.v1_create_webhook_request_base import (
    V1CreateWebhookRequestBase,
)
from hatchet_sdk.clients.rest.models.v1_create_webhook_request_basic_auth import (
    V1CreateWebhookRequestBasicAuth,
)
from hatchet_sdk.clients.rest.models.v1_create_webhook_request_hmac import (
    V1CreateWebhookRequestHMAC,
)
from hatchet_sdk.clients.rest.models.v1_update_webhook_request import (
    V1UpdateWebhookRequest,
)
from hatchet_sdk.clients.rest.models.v1_webhook import V1Webhook
from hatchet_sdk.clients.rest.models.v1_webhook_api_key_auth import V1WebhookAPIKeyAuth
from hatchet_sdk.clients.rest.models.v1_webhook_auth_type import V1WebhookAuthType
from hatchet_sdk.clients.rest.models.v1_webhook_basic_auth import V1WebhookBasicAuth
from hatchet_sdk.clients.rest.models.v1_webhook_hmac_auth import V1WebhookHMACAuth
from hatchet_sdk.clients.rest.models.v1_webhook_list import V1WebhookList
from hatchet_sdk.clients.rest.models.v1_webhook_source_name import V1WebhookSourceName
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.v1.api_client import BaseRestClient


class CreateWebhookRequest(V1CreateWebhookRequestBase):
    auth: V1WebhookBasicAuth | V1WebhookAPIKeyAuth | V1WebhookHMACAuth

    def _to_api_payload(self) -> V1CreateWebhookRequest:
        payload = self.model_dump(by_alias=True, exclude_none=True)
        payload["auth"] = self.auth.model_dump(by_alias=True)
        request_payload: (
            V1CreateWebhookRequestBasicAuth
            | V1CreateWebhookRequestAPIKey
            | V1CreateWebhookRequestHMAC
            | None
        ) = None
        if isinstance(self.auth, V1WebhookBasicAuth):
            payload["authType"] = V1WebhookAuthType.BASIC
            request_payload = V1CreateWebhookRequestBasicAuth.from_dict(payload)
        elif isinstance(self.auth, V1WebhookAPIKeyAuth):
            payload["authType"] = V1WebhookAuthType.API_KEY
            request_payload = V1CreateWebhookRequestAPIKey.from_dict(payload)
        else:
            payload["authType"] = V1WebhookAuthType.HMAC
            request_payload = V1CreateWebhookRequestHMAC.from_dict(payload)
        if request_payload is None:
            raise ValueError("failed to build create webhook request from payload")
        return V1CreateWebhookRequest(request_payload)


class WebhooksClient(BaseRestClient):
    """
    The webhooks client provides methods for managing incoming webhooks in Hatchet.

    Webhooks allow external systems to trigger Hatchet workflows by sending HTTP
    requests to dedicated endpoints. This enables real-time integration with
    third-party services like GitHub, Stripe, Slack, or any system that can send
    webhook events.
    """

    def _wa(self, client: ApiClient) -> WebhookApi:
        return WebhookApi(client)

    async def aio_list(
        self,
        limit: int | None = None,
        offset: int | None = None,
        webhook_names: list[str] | None = None,
        source_names: list[V1WebhookSourceName] | None = None,
    ) -> V1WebhookList:
        return await asyncio.to_thread(
            self.list, limit, offset, webhook_names, source_names
        )

    def list(
        self,
        limit: int | None = None,
        offset: int | None = None,
        webhook_names: list[str] | None = None,
        source_names: list[V1WebhookSourceName] | None = None,
    ) -> V1WebhookList:
        with self.client() as client:
            v1_webhook_list = tenacity_retry(
                self._wa(client).v1_webhook_list, self.client_config.tenacity
            )
            return v1_webhook_list(
                tenant=self.tenant_id,
                limit=limit,
                offset=offset,
                webhook_names=webhook_names,
                source_names=source_names,
            )

    def get(self, webhook_name: str) -> V1Webhook:
        with self.client() as client:
            v1_webhook_get = tenacity_retry(
                self._wa(client).v1_webhook_get, self.client_config.tenacity
            )
            return v1_webhook_get(
                tenant=self.tenant_id,
                v1_webhook=webhook_name,
            )

    async def aio_get(self, webhook_name: str) -> V1Webhook:
        return await asyncio.to_thread(self.get, webhook_name)

    def create(
        self,
        source_name: V1WebhookSourceName,
        name: str,
        event_key_expression: str,
        auth: V1WebhookBasicAuth | V1WebhookAPIKeyAuth | V1WebhookHMACAuth,
        scope_expression: str | None = None,
        static_payload: dict[str, Any] | None = None,
    ) -> V1Webhook:
        validated_payload = CreateWebhookRequest(
            sourceName=source_name,
            name=name,
            eventKeyExpression=event_key_expression,
            scopeExpression=scope_expression,
            staticPayload=static_payload,
            auth=auth,
        )
        with self.client() as client:
            return self._wa(client).v1_webhook_create(
                tenant=self.tenant_id,
                v1_create_webhook_request=validated_payload._to_api_payload(),
            )

    async def aio_create(
        self,
        source_name: V1WebhookSourceName,
        name: str,
        event_key_expression: str,
        auth: V1WebhookBasicAuth | V1WebhookAPIKeyAuth | V1WebhookHMACAuth,
        scope_expression: str | None = None,
        static_payload: dict[str, Any] | None = None,
    ) -> V1Webhook:
        return await asyncio.to_thread(
            self.create,
            source_name,
            name,
            event_key_expression,
            auth,
            scope_expression,
            static_payload,
        )

    def update(
        self,
        webhook_name: str,
        event_key_expression: str | None = None,
        scope_expression: str | None = None,
        static_payload: dict[str, Any] | None = None,
    ) -> V1Webhook:
        with self.client() as client:
            return self._wa(client).v1_webhook_update(
                tenant=self.tenant_id,
                v1_webhook=webhook_name,
                v1_update_webhook_request=V1UpdateWebhookRequest(
                    eventKeyExpression=event_key_expression,
                    scopeExpression=scope_expression,
                    staticPayload=static_payload,
                ),
            )

    async def aio_update(
        self,
        webhook_name: str,
        event_key_expression: str | None = None,
        scope_expression: str | None = None,
        static_payload: dict[str, Any] | None = None,
    ) -> V1Webhook:
        return await asyncio.to_thread(
            self.update,
            webhook_name,
            event_key_expression,
            scope_expression,
            static_payload,
        )

    def delete(self, webhook_name: str) -> V1Webhook:
        with self.client() as client:
            return self._wa(client).v1_webhook_delete(
                tenant=self.tenant_id,
                v1_webhook=webhook_name,
            )

    async def aio_delete(self, webhook_name: str) -> V1Webhook:
        return await asyncio.to_thread(self.delete, webhook_name)

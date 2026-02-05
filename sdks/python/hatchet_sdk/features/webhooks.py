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
from hatchet_sdk.clients.rest.models.v1_webhook_basic_auth import V1WebhookBasicAuth
from hatchet_sdk.clients.rest.models.v1_webhook_hmac_algorithm import (
    V1WebhookHMACAlgorithm,
)
from hatchet_sdk.clients.rest.models.v1_webhook_hmac_auth import V1WebhookHMACAuth
from hatchet_sdk.clients.rest.models.v1_webhook_hmac_encoding import (
    V1WebhookHMACEncoding,
)
from hatchet_sdk.clients.rest.models.v1_webhook_list import V1WebhookList
from hatchet_sdk.clients.rest.models.v1_webhook_source_name import V1WebhookSourceName
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.v1.api_client import BaseRestClient


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

    async def aio_list(
        self,
        limit: int | None = None,
        offset: int | None = None,
        webhook_names: list[str] | None = None,
        source_names: list[V1WebhookSourceName] | None = None,
    ) -> V1WebhookList:
        return await asyncio.to_thread(self.list, limit, offset, webhook_names, source_names)

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

    def create_with_basic_auth(
        self,
        name: str,
        event_key_expression: str,
        username: str,
        password: str,
        source_name: V1WebhookSourceName = V1WebhookSourceName.GENERIC,
        scope_expression: str | None = None,
        static_payload: dict[str, Any] | None = None,
    ) -> V1Webhook:
        with self.client() as client:
            return self._wa(client).v1_webhook_create(
                tenant=self.tenant_id,
                v1_create_webhook_request=V1CreateWebhookRequest(
                    V1CreateWebhookRequestBasicAuth(
                        sourceName=source_name,
                        name=name,
                        eventKeyExpression=event_key_expression,
                        scopeExpression=scope_expression,
                        staticPayload=static_payload,
                        authType="BASIC",
                        auth=V1WebhookBasicAuth(
                            username=username,
                            password=password,
                        ),
                    )
                ),
            )

    async def aio_create_with_basic_auth(
        self,
        name: str,
        event_key_expression: str,
        username: str,
        password: str,
        source_name: V1WebhookSourceName = V1WebhookSourceName.GENERIC,
        scope_expression: str | None = None,
        static_payload: dict[str, Any] | None = None,
    ) -> V1Webhook:
        return await asyncio.to_thread(
            self.create_with_basic_auth,
            name,
            event_key_expression,
            username,
            password,
            source_name,
            scope_expression,
            static_payload,
        )

    def create_with_api_key(
        self,
        name: str,
        event_key_expression: str,
        header_name: str,
        api_key: str,
        source_name: V1WebhookSourceName = V1WebhookSourceName.GENERIC,
        scope_expression: str | None = None,
        static_payload: dict[str, Any] | None = None,
    ) -> V1Webhook:
        with self.client() as client:
            return self._wa(client).v1_webhook_create(
                tenant=self.tenant_id,
                v1_create_webhook_request=V1CreateWebhookRequest(
                    V1CreateWebhookRequestAPIKey(
                        sourceName=source_name,
                        name=name,
                        eventKeyExpression=event_key_expression,
                        scopeExpression=scope_expression,
                        staticPayload=static_payload,
                        authType="API_KEY",
                        auth=V1WebhookAPIKeyAuth(
                            headerName=header_name,
                            apiKey=api_key,
                        ),
                    )
                ),
            )

    async def aio_create_with_api_key(
        self,
        name: str,
        event_key_expression: str,
        header_name: str,
        api_key: str,
        source_name: V1WebhookSourceName = V1WebhookSourceName.GENERIC,
        scope_expression: str | None = None,
        static_payload: dict[str, Any] | None = None,
    ) -> V1Webhook:
        return await asyncio.to_thread(
            self.create_with_api_key,
            name,
            event_key_expression,
            header_name,
            api_key,
            source_name,
            scope_expression,
            static_payload,
        )

    def create_with_hmac(
        self,
        name: str,
        event_key_expression: str,
        signature_header_name: str,
        signing_secret: str,
        algorithm: V1WebhookHMACAlgorithm = V1WebhookHMACAlgorithm.SHA256,
        encoding: V1WebhookHMACEncoding = V1WebhookHMACEncoding.HEX,
        source_name: V1WebhookSourceName = V1WebhookSourceName.GENERIC,
        scope_expression: str | None = None,
        static_payload: dict[str, Any] | None = None,
    ) -> V1Webhook:
        with self.client() as client:
            return self._wa(client).v1_webhook_create(
                tenant=self.tenant_id,
                v1_create_webhook_request=V1CreateWebhookRequest(
                    V1CreateWebhookRequestHMAC(
                        sourceName=source_name,
                        name=name,
                        eventKeyExpression=event_key_expression,
                        scopeExpression=scope_expression,
                        staticPayload=static_payload,
                        authType="HMAC",
                        auth=V1WebhookHMACAuth(
                            signatureHeaderName=signature_header_name,
                            signingSecret=signing_secret,
                            algorithm=algorithm,
                            encoding=encoding,
                        ),
                    )
                ),
            )

    async def aio_create_with_hmac(
        self,
        name: str,
        event_key_expression: str,
        signature_header_name: str,
        signing_secret: str,
        algorithm: V1WebhookHMACAlgorithm = V1WebhookHMACAlgorithm.SHA256,
        encoding: V1WebhookHMACEncoding = V1WebhookHMACEncoding.HEX,
        source_name: V1WebhookSourceName = V1WebhookSourceName.GENERIC,
        scope_expression: str | None = None,
        static_payload: dict[str, Any] | None = None,
    ) -> V1Webhook:
        return await asyncio.to_thread(
            self.create_with_hmac,
            name,
            event_key_expression,
            signature_header_name,
            signing_secret,
            algorithm,
            encoding,
            source_name,
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
                    eventKeyExpression=event_key_expression or "",
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

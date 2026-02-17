import {
  V1CreateWebhookRequest,
  V1CreateWebhookRequestBase,
  V1UpdateWebhookRequest,
  V1Webhook,
  V1WebhookAuthType,
  V1WebhookList,
  V1WebhookSourceName,
  V1WebhookAPIKeyAuth,
  V1WebhookBasicAuth,
  V1WebhookHMACAuth,
} from '@hatchet/clients/rest/generated/data-contracts';
import { HatchetClient } from '../client';

export type CreateWebhookOptions = V1CreateWebhookRequestBase & {
  auth: V1WebhookBasicAuth | V1WebhookAPIKeyAuth | V1WebhookHMACAuth;
};

function getAuthType(
  auth: V1WebhookBasicAuth | V1WebhookAPIKeyAuth | V1WebhookHMACAuth
): V1WebhookAuthType {
  if ('username' in auth && 'password' in auth) return V1WebhookAuthType.BASIC;
  if ('headerName' in auth && 'apiKey' in auth) return V1WebhookAuthType.API_KEY;
  if (
    'signingSecret' in auth &&
    'signatureHeaderName' in auth &&
    'algorithm' in auth &&
    'encoding' in auth
  ) {
    return V1WebhookAuthType.HMAC;
  }
  throw new Error('Invalid webhook auth');
}

function toCreateWebhookRequest(options: CreateWebhookOptions): V1CreateWebhookRequest {
  const { auth, ...base } = options;
  const authType = getAuthType(auth);
  return { ...base, authType, auth } as V1CreateWebhookRequest;
}

/**
 * Client for managing incoming webhooks in Hatchet.
 *
 * Webhooks allow external systems to trigger Hatchet workflows by sending HTTP
 * requests to dedicated endpoints. This enables real-time integration with
 * third-party services like GitHub, Stripe, Slack, or any system that can send
 * webhook events.
 */
export class WebhooksClient {
  api: HatchetClient['api'];
  tenantId: string;

  constructor(client: HatchetClient) {
    this.api = client.api;
    this.tenantId = client.tenantId;
  }

  async list(options?: {
    limit?: number;
    offset?: number;
    webhookNames?: string[];
    sourceNames?: V1WebhookSourceName[];
  }): Promise<V1WebhookList> {
    const response = await this.api.v1WebhookList(this.tenantId, {
      limit: options?.limit,
      offset: options?.offset,
      webhookNames: options?.webhookNames,
      sourceNames: options?.sourceNames,
    });
    return response.data;
  }

  async get(webhookName: string): Promise<V1Webhook> {
    const response = await this.api.v1WebhookGet(this.tenantId, webhookName);
    return response.data;
  }

  async create(request: CreateWebhookOptions): Promise<V1Webhook> {
    const payload = toCreateWebhookRequest(request);
    const response = await this.api.v1WebhookCreate(this.tenantId, payload);
    return response.data;
  }

  async update(
    webhookName: string,
    options: Partial<V1UpdateWebhookRequest> = {}
  ): Promise<V1Webhook> {
    const response = await this.api.v1WebhookUpdate(this.tenantId, webhookName, {
      eventKeyExpression: options.eventKeyExpression,
      scopeExpression: options.scopeExpression,
      staticPayload: options.staticPayload,
    });
    return response.data;
  }

  async delete(webhookName: string): Promise<V1Webhook> {
    const response = await this.api.v1WebhookDelete(this.tenantId, webhookName);
    return response.data;
  }
}

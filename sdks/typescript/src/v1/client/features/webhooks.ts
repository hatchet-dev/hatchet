import {
  V1CreateWebhookRequest,
  V1UpdateWebhookRequest,
  V1Webhook,
  V1WebhookList,
  V1WebhookSourceName,
} from '@hatchet/clients/rest/generated/data-contracts';
import { HatchetClient } from '../client';

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

  async create(request: V1CreateWebhookRequest): Promise<V1Webhook> {
    const response = await this.api.v1WebhookCreate(this.tenantId, request);
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

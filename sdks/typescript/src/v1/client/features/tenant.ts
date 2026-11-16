import { Tenant } from '@hatchet/clients/rest/generated/data-contracts';
import type { RestApi, RestClientDeps } from '@hatchet/clients/rest/rest-api';

/**
 * Client for managing Tenants
 */
export class TenantClient {
  api: RestApi;
  tenantId: string;

  constructor(client: RestClientDeps) {
    this.api = client.api;
    this.tenantId = client.tenantId;
  }

  /**
   * Retrieves the current tenant.
   * @returns The Tenant object.
   */
  async get(): Promise<Tenant> {
    return (await this.api.tenantGet(this.tenantId)).data;
  }
}

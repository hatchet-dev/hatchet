import { Tenant } from '@hatchet/clients/rest/generated/data-contracts';
import { HatchetClient } from '../client';

/**
 * Client for managing Tenants
 */
export class TenantClient {
  api: HatchetClient['api'];
  tenantId: string;

  constructor(client: HatchetClient) {
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

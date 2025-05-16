import { HatchetClient } from '../client';
/**
 * The filters client is a client for interacting with Hatchet's filters API.
 */
export class FiltersClient {
  tenantId: string;
  api: HatchetClient['api'];

  constructor(client: HatchetClient) {
    this.tenantId = client.tenantId;
    this.api = client.api;
  }

  async list(
    opts?: Parameters<typeof this.api.v1FilterList>[1]
  ) {
    const { data } = await this.api.v1FilterList(this.tenantId, opts);
    return data;
  }

  async get(
    filterId: Parameters<typeof this.api.v1FilterGet>[1]
  ) {
    const { data } = await this.api.v1FilterGet(this.tenantId, filterId);
    return data;
  }

  async create(
    opts: Parameters<typeof this.api.v1FilterCreate>[1]
  ) {
    const { data } = await this.api.v1FilterCreate(this.tenantId, opts);
    return data;
  }

  async delete(
    filterId: Parameters<typeof this.api.v1FilterDelete>[1]
  ) {
    const { data } = await this.api.v1FilterDelete(this.tenantId, filterId);
    return data;
  }
}

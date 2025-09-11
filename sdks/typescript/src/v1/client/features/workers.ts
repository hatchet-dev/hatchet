import { HatchetClient } from '../client';

/**
 * WorkersClient is used to list and manage workers
 */
export class WorkersClient {
  api: HatchetClient['api'];
  tenantId: string;

  constructor(client: HatchetClient) {
    this.api = client.api;
    this.tenantId = client.tenantId;
  }

  async get(workerId: string) {
    const { data } = await this.api.workerGet(workerId);
    return data;
  }

  async list() {
    const { data } = await this.api.workerList(this.tenantId);
    return data;
  }

  async isPaused(workerId: string) {
    const wf = await this.get(workerId);
    return wf.status === 'PAUSED';
  }

  async pause(workerId: string) {
    const { data } = await this.api.workerUpdate(workerId, {
      isPaused: true,
    });
    return data;
  }

  async unpause(workerId: string) {
    const { data } = await this.api.workerUpdate(workerId, {
      isPaused: false,
    });
    return data;
  }
}

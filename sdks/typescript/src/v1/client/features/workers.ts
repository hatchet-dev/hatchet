import { HatchetClient } from '../client';

/**
 * The workers client is a client for managing workers programmatically within Hatchet.
 */
export class WorkersClient {
  api: HatchetClient['api'];
  tenantId: string;

  constructor(client: HatchetClient) {
    this.api = client.api;
    this.tenantId = client.tenantId;
  }

  /**
   * Get a worker by its ID.
   * @param workerId - The ID of the worker to get.
   * @returns A promise that resolves to the worker.
   */
  async get(workerId: string) {
    const { data } = await this.api.workerGet(workerId);
    return data;
  }

  /**
   * List all workers in the tenant.
   * @returns A promise that resolves to the list of workers.
   */
  async list() {
    const { data } = await this.api.workerList(this.tenantId);
    return data;
  }

  /**
   * Check if a worker is paused.
   * @param workerId - The ID of the worker to check.
   * @returns A promise that resolves to true if the worker is paused, false otherwise.
   */
  async isPaused(workerId: string) {
    const wf = await this.get(workerId);
    return wf.status === 'PAUSED';
  }

  /**
   * Pause a worker.
   * @param workerId - The ID of the worker to pause.
   * @returns A promise that resolves to the paused worker.
   */
  async pause(workerId: string) {
    const { data } = await this.api.workerUpdate(workerId, {
      isPaused: true,
    });
    return data;
  }

  /**
   * Unpause a worker.
   * @param workerId - The ID of the worker to unpause.
   * @returns A promise that resolves to the unpaused worker.
   */
  async unpause(workerId: string) {
    const { data } = await this.api.workerUpdate(workerId, {
      isPaused: false,
    });
    return data;
  }
}

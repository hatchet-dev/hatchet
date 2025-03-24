import { HatchetClient } from '../client';

/**
 * WorkersClient is used to list and manage workers
 */
export class WorkersClient {
  api: HatchetClient['api'];

  constructor(client: HatchetClient) {
    this.api = client.api;
  }
}

import { HatchetClient } from '../client';

/**
 * RunsClient is used to list and manage runs
 */
export class RunsClient {
  api: HatchetClient['api'];

  constructor(client: HatchetClient) {
    this.api = client.api;
  }
}

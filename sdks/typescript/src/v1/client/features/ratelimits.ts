import { HatchetClient } from '../client';
/**
 * RatelimitsClient is used to manage rate limits for the Hatchet
 */
export class RatelimitsClient {
  api: HatchetClient['api'];

  constructor(client: HatchetClient) {
    this.api = client.api;
  }
}

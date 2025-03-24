import { HatchetClient } from '../client';
/**
 * MetricsClient is used to get metrics for workflows
 */
export class MetricsClient {
  api: HatchetClient['api'];

  constructor(client: HatchetClient) {
    this.api = client.api;
  }
}

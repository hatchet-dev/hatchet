import { HatchetClient } from '../client';
/**
 * WorkflowsClient is used to list and manage workflows
 */
export class WorkflowsClient {
  api: HatchetClient['api'];

  constructor(client: HatchetClient) {
    this.api = client.api;
  }
}

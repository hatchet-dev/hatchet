import { LegacyHatchetClient } from '@hatchet/clients/hatchet-client';
import { MetricsClient } from './features/metrics';
import { RunsClient } from './features/runs';
import { WorkersClient } from './features/workers';
import { WorkflowsClient } from './features/workflows';
import { AdminClient } from './admin';

export interface IHatchetClient {
  _v0: LegacyHatchetClient;

  metrics: MetricsClient;
  runs: RunsClient;
  workflows: WorkflowsClient;
  workers: WorkersClient;

  admin: AdminClient;
}

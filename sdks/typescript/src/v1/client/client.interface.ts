import { InternalHatchetClient } from '@hatchet/clients/hatchet-client';
import { MetricsClient } from './features/metrics';
import { RunsClient } from './features/runs';
import { WorkersClient } from './features/workers';
import { WorkflowsClient } from './features/workflows';

export interface IHatchetClient {
  v0: InternalHatchetClient;

  metrics: MetricsClient;
  runs: RunsClient;
  workflows: WorkflowsClient;
  workers: WorkersClient;
}

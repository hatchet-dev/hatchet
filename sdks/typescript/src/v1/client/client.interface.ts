import { LegacyHatchetClient } from '@hatchet/clients/hatchet-client';
import { MetricsClient } from './features/metrics';
import { RunsClient } from './features/runs';
import { WorkersClient } from './features/workers';
import { WorkflowsClient } from './features/workflows';
import { AdminClient } from './admin';
import { ScheduleClient } from './features/schedules';
import { CronClient } from './features/crons';
import { CELClient } from './features/cel';

export interface IHatchetClient {
  _v0: LegacyHatchetClient;

  cel: CELClient;
  metrics: MetricsClient;
  runs: RunsClient;
  workflows: WorkflowsClient;
  workers: WorkersClient;

  scheduled: ScheduleClient;
  crons: CronClient;
  admin: AdminClient;
}

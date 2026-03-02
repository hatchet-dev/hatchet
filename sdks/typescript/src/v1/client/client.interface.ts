import { EventClient } from '@hatchet/clients/event/event-client';
import { DispatcherClient } from '@hatchet/clients/dispatcher/dispatcher-client';
import { Logger } from '@hatchet/util/logger';
import { LegacyHatchetClient } from '../../legacy/legacy-client';
import { MetricsClient } from './features/metrics';
import { RunsClient } from './features/runs';
import { WorkersClient } from './features/workers';
import { WorkflowsClient } from './features/workflows';
import { AdminClient } from './admin';
import { ScheduleClient } from './features/schedules';
import { CronClient } from './features/crons';
import { CELClient } from './features/cel';

export interface IHatchetClient {
  /** @deprecated v0 client will be removed in a future release, please upgrade to v1 */
  v0: LegacyHatchetClient;
  cel: CELClient;
  dispatcher: DispatcherClient;
  events: EventClient;
  logger: Logger;
  metrics: MetricsClient;
  runs: RunsClient;
  workflows: WorkflowsClient;
  workers: WorkersClient;

  scheduled: ScheduleClient;
  crons: CronClient;
  admin: AdminClient;
}

import { EventClient } from '@hatchet/clients/event/event-client';
import { DispatcherClient } from '@hatchet/clients/dispatcher/dispatcher-client';
import { Logger } from '@hatchet/util/logger';
import { LegacyHatchetClient } from '../../legacy/legacy-client';
import {
  CronClient,
  MetricsClient,
  RunsClient,
  ScheduleClient,
  WebhooksClient,
  WorkersClient,
  WorkflowsClient,
  CELClient,
  LogsClient,
  TenantClient,
  FiltersClient,
  RatelimitsClient,
} from './features';
import { AdminClient } from './admin';

export interface IHatchetClient {
  /** @deprecated v0 client will be removed in a future release, please upgrade to v1 */
  v0: LegacyHatchetClient;
  cel: CELClient;
  dispatcher: DispatcherClient;
  events: EventClient;
  logger: Logger;
  logs: LogsClient;
  metrics: MetricsClient;
  runs: RunsClient;
  workflows: WorkflowsClient;
  workers: WorkersClient;
  webhooks: WebhooksClient;
  tenant: TenantClient;
  filters: FiltersClient; 
  ratelimits: RatelimitsClient;
  scheduled: ScheduleClient;
  crons: CronClient;
  admin: AdminClient;
}

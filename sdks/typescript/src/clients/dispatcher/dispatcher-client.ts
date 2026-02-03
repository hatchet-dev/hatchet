import { Channel, ClientFactory } from 'nice-grpc';
import {
  DispatcherClient as PbDispatcherClient,
  DispatcherDefinition,
  StepActionEvent,
  GroupKeyActionEvent,
  OverridesData,
  DeepPartial,
  WorkerLabels as PbWorkerAffinityConfig,
  SDKS,
  RuntimeInfo,
} from '@hatchet/protoc/dispatcher';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import HatchetError from '@util/errors/hatchet-error';
import { Logger } from '@hatchet/util/logger';

import { retrier } from '@hatchet/util/retrier';
import { HATCHET_VERSION } from '@hatchet/version';
import { ActionListener } from './action-listener';

export type WorkerLabels = Record<string, string | number | undefined>;

interface GetActionListenerOptions {
  workerName: string;
  services: string[];
  actions: string[];
  slots?: number;
  durableSlots?: number;
  labels: Record<string, string | number | undefined>;
}

type StepActionEventInput = StepActionEvent & {
  /** @deprecated use taskId */
  stepId?: string;
  /** @deprecated use taskRunId */
  stepRunId?: string;
};

export class DispatcherClient {
  config: ClientConfig;
  client: PbDispatcherClient;
  logger: Logger;

  constructor(config: ClientConfig, channel: Channel, factory: ClientFactory) {
    this.config = config;
    this.client = factory.create(DispatcherDefinition, channel);
    this.logger = config.logger(`Dispatcher`, config.log_level);
  }

  getRuntimeInfo(): RuntimeInfo {
    return {
      sdkVersion: HATCHET_VERSION,
      language: SDKS.TYPESCRIPT,
      languageVersion: process.version,
      os: process.platform,
    };
  }

  async getActionListener(options: GetActionListenerOptions) {
    // Register the worker
    const { slots, ...rest } = options;
    const registration = await this.client.register({
      ...rest,
      slots,
      labels: options.labels ? mapLabels(options.labels) : undefined,
      runtimeInfo: this.getRuntimeInfo(),
    });

    return new ActionListener(this, registration.workerId);
  }

  async sendStepActionEvent(in_: StepActionEventInput) {
    const { taskId, taskRunExternalId, ...rest } = in_;
    const event: StepActionEvent = {
      ...rest,
      taskId: taskId ?? '',
      taskRunExternalId: taskRunExternalId ?? '',
    };

    try {
      return await retrier(async () => this.client.sendStepActionEvent(event), this.logger);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  async sendGroupKeyActionEvent(in_: GroupKeyActionEvent) {
    try {
      return await retrier(async () => this.client.sendGroupKeyActionEvent(in_), this.logger);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  async putOverridesData(in_: DeepPartial<OverridesData>) {
    return retrier(async () => this.client.putOverridesData(in_), this.logger).catch((e) => {
      this.logger.warn(`Could not put overrides data: ${e.message}`);
    });
  }

  async refreshTimeout(incrementTimeoutBy: string, taskRunExternalId: string) {
    try {
      return this.client.refreshTimeout({
        taskRunExternalId,
        incrementTimeoutBy,
      });
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  async upsertWorkerLabels(workerId: string, labels: WorkerLabels) {
    try {
      return await this.client.upsertWorkerLabels({
        workerId,
        labels: mapLabels(labels),
      });
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }
}

function mapLabels(in_: WorkerLabels): Record<string, PbWorkerAffinityConfig> {
  return Object.entries(in_).reduce<Record<string, PbWorkerAffinityConfig>>(
    (acc, [key, value]) => ({
      ...acc,
      [key]: {
        strValue: typeof value === 'string' ? value : undefined,
        intValue: typeof value === 'number' ? value : undefined,
      } as PbWorkerAffinityConfig,
    }),
    {} as Record<string, PbWorkerAffinityConfig>
  );
}

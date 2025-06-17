/* eslint-disable no-underscore-dangle */
/* eslint-disable max-classes-per-file */
import {
  RunListenerClient,
  StepRunEvent,
} from '@hatchet/clients/listeners/run-listener/child-listener-client';
import { Status } from 'nice-grpc';
import { RunsClient } from '@hatchet/v1';
import { WorkflowRunEventType } from '../protoc/dispatcher';

type EventualWorkflowRunId =
  | string
  | Promise<string>
  | Promise<{
      workflowRunId: string;
    }>;

export class DedupeViolationErr extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'DedupeViolationErr';
  }
}

async function getWorkflowRunId(workflowRunId: EventualWorkflowRunId): Promise<string> {
  if (typeof workflowRunId === 'string') {
    return workflowRunId;
  }

  if (workflowRunId instanceof Promise) {
    try {
      const resolved = await workflowRunId;
      if (typeof resolved === 'string') {
        return resolved;
      }

      return resolved.workflowRunId;
    } catch (e: any) {
      if (e.code && e.code === Status.ALREADY_EXISTS) {
        throw new DedupeViolationErr(e.details);
      }

      throw e;
    }
  }

  throw new Error('Invalid workflowRunId: must be a string or a promise');
}

export default class WorkflowRunRef<T> {
  workflowRunId: EventualWorkflowRunId;
  parentWorkflowRunId?: string;
  private client: RunListenerClient;
  private runs: RunsClient | undefined;
  _standaloneTaskName?: string;

  constructor(
    workflowRunId:
      | string
      | Promise<string>
      | Promise<{
          workflowRunId: string;
        }>,
    client: RunListenerClient,
    runsClient?: RunsClient,
    parentWorkflowRunId?: string,
    standaloneTaskName?: string
  ) {
    this.workflowRunId = workflowRunId;
    this.parentWorkflowRunId = parentWorkflowRunId;
    this.client = client;
    this.runs = runsClient;
    this._standaloneTaskName = standaloneTaskName;
  }

  // TODO docstrings
  get runId() {
    return this.getWorkflowRunId();
  }

  // @deprecated use runId
  async getWorkflowRunId(): Promise<string> {
    return getWorkflowRunId(this.workflowRunId);
  }

  async stream(): Promise<AsyncGenerator<StepRunEvent, void, unknown>> {
    const workflowRunId = await getWorkflowRunId(this.workflowRunId);
    return this.client.stream(workflowRunId);
  }

  get output(): Promise<T> {
    return this.result();
  }

  /**
   * @alias output
   * @deprecated use output
   */
  async result(): Promise<T> {
    const workflowRunId = await getWorkflowRunId(this.workflowRunId);

    const streamable = await this.client.get(workflowRunId);

    return new Promise<T>((resolve, reject) => {
      (async () => {
        for await (const event of streamable.stream()) {
          if (event.eventType === WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_FINISHED) {
            if (event.results.some((r) => r.error !== undefined)) {
              reject(event.results);
              return;
            }

            if (event.results.length === 0) {
              const data = await this.client.api.workflowRunGetShape(
                this.client.config.tenant_id,
                event.workflowRunId
              );

              const mostRecentJobRun = data.data.jobRuns?.[0];

              if (!mostRecentJobRun) {
                reject(new Error('No job runs found'));
                return;
              }

              const outputs: { [readableStepName: string]: any } = {};

              mostRecentJobRun.stepRuns?.forEach((stepRun) => {
                const readable = mostRecentJobRun.job?.steps?.find(
                  (step) => step.metadata.id === stepRun.stepId
                );
                const readableStepName = `${readable?.readableId}`;
                try {
                  outputs[readableStepName] = JSON.parse(stepRun.output || '{}');
                } catch (error) {
                  outputs[readableStepName] = stepRun.output;
                }
              });

              if (!this._standaloneTaskName) {
                resolve(outputs as T);
                return;
              }

              resolve(outputs[this._standaloneTaskName] as T);
              return;
            }

            const result = event.results.reduce(
              (acc, r) => ({
                ...acc,
                [r.stepReadableId]: JSON.parse(r.output || '{}'),
              }),
              {} as T
            );

            if (!this._standaloneTaskName) {
              resolve(result);
              return;
            }

            resolve((result as any)[this._standaloneTaskName] as T);
            return;
          }
        }
      })();
    });
  }

  async toJSON(): Promise<string> {
    return JSON.stringify({
      workflowRunId: await this.workflowRunId,
    });
  }

  async cancel() {
    if (!this.runs) {
      throw new Error('cancel is a v1 only feature, please upgrade your sdk');
    }

    const workflowRunId = await getWorkflowRunId(this.workflowRunId);
    await this.runs.cancel({
      ids: [workflowRunId],
    });
  }

  async replay() {
    if (!this.runs) {
      throw new Error('replay is a v1 only feature, please upgrade your sdk');
    }

    const workflowRunId = await getWorkflowRunId(this.workflowRunId);
    await this.runs.replay({
      ids: [workflowRunId],
    });
  }
}

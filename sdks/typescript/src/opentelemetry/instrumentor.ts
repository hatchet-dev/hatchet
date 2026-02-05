/**
 * Hatchet OpenTelemetry Instrumentor
 *
 * This module provides automatic instrumentation for Hatchet SDK operations
 * including workflow runs, event pushes, and step executions.
 *
 * The instrumentor follows the OpenTelemetry instrumentation pattern,
 * patching module prototypes to automatically instrument all instances.
 */

try {
  require.resolve('@opentelemetry/api');
  require.resolve('@opentelemetry/instrumentation');
} catch {
  throw new Error(
    'To use HatchetInstrumentor, you must install OpenTelemetry packages: npm install @opentelemetry/api @opentelemetry/instrumentation'
  );
}

// eslint-disable-next-line @typescript-eslint/no-require-imports
const otelApi = require('@opentelemetry/api') as typeof import('@opentelemetry/api');
const { trace, context, propagation, SpanKind, SpanStatusCode, diag } = otelApi;

// eslint-disable-next-line @typescript-eslint/no-require-imports
const otelInstrumentation = require('@opentelemetry/instrumentation') as typeof import('@opentelemetry/instrumentation');
const { InstrumentationBase, InstrumentationNodeModuleDefinition, InstrumentationNodeModuleFile, safeExecuteInTheMiddle, isWrapped } =
  otelInstrumentation;

import type {
  Context as OtelContext,
  Span,
  Attributes,
} from '@opentelemetry/api';

import type { InstrumentationConfig } from '@opentelemetry/instrumentation';

import { HATCHET_VERSION } from '@hatchet/version';
import { Action } from '@hatchet/clients/dispatcher/action-listener';
import type { EventClient, PushEventOptions, EventWithMetadata } from '@hatchet/clients/event/event-client';
import type { AdminClient } from '@hatchet/v1/client/admin';
import type { V1Worker } from '@hatchet/v1/client/worker/worker-internal';
import { OTelAttribute } from '../util/opentelemetry';
import { OpenTelemetryConfig, DEFAULT_CONFIG } from './types';
import { ScheduledWorkflows } from '../clients/rest/generated/data-contracts';
import { ScheduleClient } from '../v1/client/features/schedules';

type HatchetInstrumentationConfig = OpenTelemetryConfig & InstrumentationConfig;
type Carrier = Record<string, string>;

const INSTRUMENTOR_NAME = '@hatchet-dev/typescript-sdk';
const SUPPORTED_VERSIONS = ['>=1.9.0'];

function extractContext(carrier: Carrier | undefined | null): OtelContext {
  return propagation.extract(context.active(), carrier ?? {});
}

function injectContext(carrier: Carrier): void {
  propagation.inject(context.active(), carrier);
}

function parseAdditionalMetadata(action: Action): Record<string, any> | undefined {
  if (!action.additionalMetadata) {
    return undefined;
  }
  try {
    return JSON.parse(action.additionalMetadata);
  } catch {
    return undefined;
  }
}

function getActionOtelAttributes(
  action: Action,
  excludedAttributes: string[] = [],
  workerId?: string
): Attributes {
  const attributes: Attributes = {
    [OTelAttribute.TENANT_ID]: action.tenantId,
    [OTelAttribute.WORKER_ID]: workerId,
    [OTelAttribute.WORKFLOW_RUN_ID]: action.workflowRunId,
    [OTelAttribute.STEP_ID]: action.stepId,
    [OTelAttribute.STEP_RUN_ID]: action.stepRunId,
    [OTelAttribute.RETRY_COUNT]: action.retryCount,
    [OTelAttribute.PARENT_WORKFLOW_RUN_ID]: action.parentWorkflowRunId,
    [OTelAttribute.CHILD_WORKFLOW_INDEX]: action.childWorkflowIndex,
    [OTelAttribute.CHILD_WORKFLOW_KEY]: action.childWorkflowKey,
    [OTelAttribute.ACTION_PAYLOAD]: action.actionPayload,
    [OTelAttribute.WORKFLOW_NAME]: action.jobName,
    [OTelAttribute.ACTION_NAME]: action.actionId,
    [OTelAttribute.WORKFLOW_ID]: action.workflowId,
    [OTelAttribute.WORKFLOW_VERSION_ID]: action.workflowVersionId,
  };

  const filtered: Attributes = {};
  for (const [key, value] of Object.entries(attributes)) {
    if (!excludedAttributes.includes(key) && value !== undefined && value !== '') {
      filtered[key] = value;
    }
  }

  return filtered;
}


function filterAttributes(
  attributes: Record<string, any>,
  excludedAttributes: string[] = []
): Attributes {
  const filtered: Attributes = {};
  for (const [key, value] of Object.entries(attributes)) {
    if (
      !excludedAttributes.includes(key) &&
      value !== undefined &&
      value !== null &&
      value !== '' &&
      value !== '{}' &&
      value !== '[]'
    ) {
      filtered[`hatchet.${key}`] = typeof value === 'object' ? JSON.stringify(value) : value;
    }
  }
  return filtered;
}

/**
 * HatchetInstrumentor provides OpenTelemetry instrumentation for Hatchet SDK v1.
 *
 * It automatically instruments:
 * - Workflow runs (runWorkflow, runWorkflows)
 * - Scheduled workflow runs (schedules.create)
 * - Event pushes (push, bulkPush)
 * - Step executions (handleStartStepRun, handleCancelStepRun)
 *
 * Traceparent context is automatically propagated through metadata.
 *
 * The instrumentor uses the global tracer/meter providers by default.
 * Use `setTracerProvider()` and `setMeterProvider()` to configure custom providers.
 */
export class HatchetInstrumentor extends InstrumentationBase<HatchetInstrumentationConfig> {
  constructor(config: Partial<HatchetInstrumentationConfig> = {}) {
    super(INSTRUMENTOR_NAME, HATCHET_VERSION, { ...DEFAULT_CONFIG, ...config });
  }

  override setConfig(config: Partial<HatchetInstrumentationConfig> = {}): void {
    super.setConfig({ ...DEFAULT_CONFIG, ...config });
  }

  protected init(): InstanceType<typeof InstrumentationNodeModuleDefinition>[] {
    const eventClientModuleFile = new InstrumentationNodeModuleFile(
      '@hatchet-dev/typescript-sdk/clients/event/event-client.js',
      SUPPORTED_VERSIONS,
      this.patchEventClient.bind(this),
      this.unpatchEventClient.bind(this)
    );

    const adminClientModuleFile = new InstrumentationNodeModuleFile(
      '@hatchet-dev/typescript-sdk/v1/client/admin.js',
      SUPPORTED_VERSIONS,
      this.patchAdminClient.bind(this),
      this.unpatchAdminClient.bind(this)
    );

    const scheduleClientModuleFile = new InstrumentationNodeModuleFile(
      '@hatchet-dev/typescript-sdk/v1/client/features/schedules.js',
      SUPPORTED_VERSIONS,
      this.patchScheduleClient.bind(this),
      this.unpatchScheduleClient.bind(this)
    );

    const workerModuleFile = new InstrumentationNodeModuleFile(
      '@hatchet-dev/typescript-sdk/v1/client/worker/worker-internal.js',
      SUPPORTED_VERSIONS,
      this.patchWorker.bind(this),
      this.unpatchWorker.bind(this)
    );

    const moduleDefinition = new InstrumentationNodeModuleDefinition(
      INSTRUMENTOR_NAME,
      SUPPORTED_VERSIONS,
      undefined,
      undefined,
      [eventClientModuleFile, adminClientModuleFile, workerModuleFile, scheduleClientModuleFile]
    );

    return [moduleDefinition];
  }


  private patchEventClient(moduleExports: any, moduleVersion?: string): any {
    if (!moduleExports?.EventClient?.prototype) {
      diag.debug('hatchet instrumentation: EventClient not found in module exports');
      return moduleExports;
    }
    this._patchPushEvent(moduleExports.EventClient.prototype);
    this._patchBulkPushEvent(moduleExports.EventClient.prototype);

    return moduleExports;
  }

  private unpatchEventClient(moduleExports: any, moduleVersion?: string): any {
    if (!moduleExports?.EventClient?.prototype) {
      return moduleExports;
    }

    if (isWrapped(moduleExports.EventClient.prototype.push)) {
      this._unwrap(moduleExports.EventClient.prototype, 'push');
    }
    if (isWrapped(moduleExports.EventClient.prototype.bulkPush)) {
      this._unwrap(moduleExports.EventClient.prototype, 'bulkPush');
    }

    return moduleExports;
  }

  private _patchPushEvent(prototype: EventClient): void {
    if (isWrapped(prototype.push)) {
      this._unwrap(prototype, 'push');
    }
    const self = this;

    this._wrap(prototype, 'push', (original: EventClient['push']) => {
      return function wrappedPush<T>(
        this: EventClient,
        type: string,
        input: T,
        options: PushEventOptions = {}
      ) {
        const attributes = filterAttributes(
          {
            [OTelAttribute.EVENT_KEY]: type,
            [OTelAttribute.ACTION_PAYLOAD]: JSON.stringify(input),
            [OTelAttribute.ADDITIONAL_METADATA]: options.additionalMetadata
              ? JSON.stringify(options.additionalMetadata)
              : undefined,
            [OTelAttribute.PRIORITY]: options.priority,
            [OTelAttribute.FILTER_SCOPE]: options.scope
          },
          self.getConfig().excludedAttributes
        );

        return self.tracer.startActiveSpan(
          'hatchet.push_event',
          {
            kind: SpanKind.PRODUCER,
            attributes,
          },
          (span: Span) => {
            const enhancedMetadata: Carrier = { ...(options.additionalMetadata ?? {}) };
            injectContext(enhancedMetadata);

            const enhancedOptions: PushEventOptions = {
              ...options,
              additionalMetadata: enhancedMetadata,
            };

            const result = original.call(this, type, input, enhancedOptions);

            return result.finally(() => {
              span.end();
            });
          }
        );
      };
    });
  }

  private _patchBulkPushEvent(prototype: EventClient): void {
    if (isWrapped(prototype.bulkPush)) {
      this._unwrap(prototype, 'bulkPush');
    }
    const self = this;

    this._wrap(prototype, 'bulkPush', (original: EventClient['bulkPush']) => {
      return function wrappedBulkPush<T>(
        this: EventClient,
        type: string,
        inputs: EventWithMetadata<T>[],
        options: PushEventOptions = {}
      ) {
        const attributes = filterAttributes(
          {
            [OTelAttribute.EVENT_KEY]: type,
            [OTelAttribute.ACTION_PAYLOAD]: JSON.stringify(inputs),
            [OTelAttribute.ADDITIONAL_METADATA]: options.additionalMetadata
              ? JSON.stringify(options.additionalMetadata)
              : undefined,
            [OTelAttribute.PRIORITY]: options.priority,
          },
          self.getConfig().excludedAttributes
        );

        return self.tracer.startActiveSpan(
          'hatchet.bulk_push_event',
          {
            kind: SpanKind.PRODUCER,
            attributes,
          },
          (span: Span) => {
            const enhancedInputs = inputs.map((input) => {
              const enhancedMetadata: Carrier = {
                ...((input.additionalMetadata as Carrier) ?? {}),
              };
              injectContext(enhancedMetadata);
              return {
                ...input,
                additionalMetadata: enhancedMetadata,
              };
            });

            const result = original.call(this, type, enhancedInputs, options);

            return result.finally(() => {
              span.end();
            });
          }
        );
      };
    });
  }

  private patchAdminClient(moduleExports: any, moduleVersion?: string): any {
    if (!moduleExports?.AdminClient?.prototype) {
      diag.debug('hatchet instrumentation: AdminClient not found in module exports');
      return moduleExports;
    }

    this._patchRunWorkflow(moduleExports.AdminClient.prototype);
    this._patchRunWorkflows(moduleExports.AdminClient.prototype);

    return moduleExports;
  }

  private unpatchAdminClient(moduleExports: any, moduleVersion?: string): any {
    if (!moduleExports?.AdminClient?.prototype) {
      return moduleExports;
    }

    if (isWrapped(moduleExports.AdminClient.prototype.runWorkflow)) {
      this._unwrap(moduleExports.AdminClient.prototype, 'runWorkflow');
    }
    if (isWrapped(moduleExports.AdminClient.prototype.runWorkflows)) {
      this._unwrap(moduleExports.AdminClient.prototype, 'runWorkflows');
    }

    return moduleExports;
  }

  private _patchRunWorkflow(prototype: AdminClient): void {
    if (isWrapped(prototype.runWorkflow)) {
      this._unwrap(prototype, 'runWorkflow');
    }
    const self = this;

    this._wrap(prototype, 'runWorkflow', (original: AdminClient['runWorkflow']) => {
      return async function wrappedRunWorkflow(
        this: AdminClient,
        workflowName: string,
        input: any,
        options?: {
          parentId?: string;
          parentStepRunId?: string;
          childIndex?: number;
          childKey?: string;
          additionalMetadata?: Record<string, string>;
          desiredWorkerId?: string;
          priority?: number;
        }
      ) {
        const attributes = filterAttributes(
          {
            [OTelAttribute.WORKFLOW_NAME]: workflowName,
            [OTelAttribute.ACTION_PAYLOAD]: JSON.stringify(input),
            [OTelAttribute.PARENT_ID]: options?.parentId,
            [OTelAttribute.PARENT_STEP_RUN_ID]: options?.parentStepRunId,
            [OTelAttribute.CHILD_INDEX]: options?.childIndex,
            [OTelAttribute.CHILD_KEY]: options?.childKey,
            [OTelAttribute.ADDITIONAL_METADATA]: options?.additionalMetadata
              ? JSON.stringify(options.additionalMetadata)
              : undefined,
            [OTelAttribute.PRIORITY]: options?.priority,
            [OTelAttribute.DESIRED_WORKER_ID]: options?.desiredWorkerId,
          },
          self.getConfig().excludedAttributes
        );

        return self.tracer.startActiveSpan(
          'hatchet.run_workflow',
          {
            kind: SpanKind.PRODUCER,
            attributes,
          },
          (span: Span) => {
            const enhancedMetadata: Carrier = { ...(options?.additionalMetadata ?? {}) };
            injectContext(enhancedMetadata);

            const enhancedOptions = {
              ...options,
              additionalMetadata: enhancedMetadata,
            };

            return original.call(this, workflowName, input, enhancedOptions)
              .catch((error: Error) => {
                span.recordException(error);
                span.setStatus({ code: SpanStatusCode.ERROR, message: error?.message });
                throw error;
              })
              .finally(() => {
                span.end();
              });
          }
        );
      } as AdminClient['runWorkflow'];
    });
  }

  private _patchRunWorkflows(prototype: AdminClient): void {
    if (isWrapped(prototype.runWorkflows)) {
      this._unwrap(prototype, 'runWorkflows');
    }
    const self = this;

    this._wrap(prototype, 'runWorkflows', (original: AdminClient['runWorkflows']) => {
      return async function wrappedRunWorkflows(
        this: AdminClient,
        workflowRuns: Array<{
          workflowName: string;
          input: any;
          options?: {
            parentId?: string;
            parentStepRunId?: string;
            childIndex?: number;
            childKey?: string;
            additionalMetadata?: Record<string, string>;
            desiredWorkerId?: string;
            priority?: number;
          };
        }>,
        batchSize?: number
      ) {
        const attributes = filterAttributes(
          {
            [OTelAttribute.WORKFLOW_NAME]: JSON.stringify(workflowRuns.map((r) => r.workflowName)),
            [OTelAttribute.ACTION_PAYLOAD]: JSON.stringify(workflowRuns),
          },
          self.getConfig().excludedAttributes
        );

        return self.tracer.startActiveSpan(
          'hatchet.run_workflows',
          {
            kind: SpanKind.PRODUCER,
            attributes,
          },
          (span: Span) => {
            const enhancedWorkflowRuns = workflowRuns.map((run) => {
              const enhancedMetadata: Carrier = { ...(run.options?.additionalMetadata ?? {}) };
              injectContext(enhancedMetadata);
              return {
                ...run,
                options: {
                  ...run.options,
                  additionalMetadata: enhancedMetadata,
                },
              };
            });

            return original.call(this, enhancedWorkflowRuns, batchSize)
              .catch((error: Error) => {
                span.recordException(error);
                span.setStatus({ code: SpanStatusCode.ERROR, message: error?.message });
                throw error;
              })
              .finally(() => {
                span.end();
              });
          }
        );
      } as AdminClient['runWorkflows'];
    });
  }

  private patchWorker(moduleExports: any, moduleVersion?: string): any {
    if (!moduleExports?.V1Worker?.prototype) {
      diag.debug('hatchet instrumentation: V1Worker not found in module exports');
      return moduleExports;
    }

    this._patchHandleStartStepRun(moduleExports.V1Worker.prototype);
    this._patchHandleCancelStepRun(moduleExports.V1Worker.prototype);

    return moduleExports;
  }

  private unpatchWorker(moduleExports: any, moduleVersion?: string): any {
    if (!moduleExports?.V1Worker?.prototype) {
      return moduleExports;
    }

    if (isWrapped(moduleExports.V1Worker.prototype.handleStartStepRun)) {
      this._unwrap(moduleExports.V1Worker.prototype, 'handleStartStepRun');
    }
    if (isWrapped(moduleExports.V1Worker.prototype.handleCancelStepRun)) {
      this._unwrap(moduleExports.V1Worker.prototype, 'handleCancelStepRun');
    }

    return moduleExports;
  }

  // IMPORTANT: Keep this wrapper's signature in sync with V1Worker.handleStartStepRun
  private _patchHandleStartStepRun(prototype: V1Worker): void {
    if (isWrapped(prototype.handleStartStepRun)) {
      this._unwrap(prototype, 'handleStartStepRun');
    }
    const self = this;

    this._wrap(prototype, 'handleStartStepRun', (original: V1Worker['handleStartStepRun']) => {
      return async function wrappedHandleStartStepRun(
        this: V1Worker,
        action: Action
      ): Promise<Error | undefined> {
        const additionalMetadata = parseAdditionalMetadata(action);
        const parentContext = extractContext(additionalMetadata);
        const attributes = getActionOtelAttributes(action, self.getConfig().excludedAttributes, this.workerId);

        let spanName = 'hatchet.start_step_run';
        if (self.getConfig().includeTaskNameInSpanName) {
          spanName += `.${action.actionId}`;
        }

        return self.tracer.startActiveSpan(
          spanName,
          {
            kind: SpanKind.CONSUMER,
            attributes,
          },
          parentContext,
          (span: Span) => {
            return original.call(this, action)
              .then((taskError: Error | undefined) => {
                if (taskError instanceof Error) {
                  span.recordException(taskError);
                  span.setStatus({ code: SpanStatusCode.ERROR, message: taskError.message });
                }
                return taskError;
              })
              .finally(() => {
                span.end();
              });
          }
        );
      };
    });
  }

  private _patchHandleCancelStepRun(prototype: V1Worker): void {
    if (isWrapped(prototype.handleCancelStepRun)) {
      this._unwrap(prototype, 'handleCancelStepRun');
    }
    const self = this;

    this._wrap(prototype, 'handleCancelStepRun', (original: V1Worker['handleCancelStepRun']) => {
      return async function wrappedHandleCancelStepRun(
        this: V1Worker,
        action: Action
      ): Promise<void> {
        const attributes: Attributes = {
          [`hatchet.${OTelAttribute.STEP_RUN_ID}`]: action.stepRunId,
        };

        return self.tracer.startActiveSpan(
          'hatchet.cancel_step_run',
          {
            kind: SpanKind.CONSUMER,
            attributes,
          },
          (span: Span) => {
            const result = original.call(this, action);

            return result.finally(() => {
              span.end();
            });
          }
        );
      };
    });
  }

  private patchScheduleClient(moduleExports: any, moduleVersion?: string): any {
    if (!moduleExports?.ScheduleClient?.prototype) {
      diag.debug('hatchet instrumentation: ScheduleClient not found in module exports');
      return moduleExports;
    }

    this._patchScheduleCreate(moduleExports.ScheduleClient.prototype);

    return moduleExports;
  }

  private unpatchScheduleClient(moduleExports: any, moduleVersion?: string): any {
    if (!moduleExports?.ScheduleClient?.prototype) {
      return moduleExports;
    }

    if (isWrapped(moduleExports.ScheduleClient.prototype.create)) {
      this._unwrap(moduleExports.ScheduleClient.prototype, 'create');
    }

    return moduleExports;
  }

  // IMPORTANT: Keep this wrapper's signature in sync with ScheduleClient.create
  private _patchScheduleCreate(prototype: ScheduleClient): void {
    if (isWrapped(prototype.create)) {
      this._unwrap(prototype, 'create');
    }
    const self = this;

    this._wrap(prototype, 'create', (original: ScheduleClient['create']) => {
      return async function wrappedCreate(
        this: ScheduleClient,
        workflow: string,
        input: any
      ): Promise<ScheduledWorkflows> {
        const triggerAtIso = input.triggerAt instanceof Date
          ? input.triggerAt.toISOString()
          : new Date(input.triggerAt).toISOString();

        const attributes = filterAttributes(
          {
            [OTelAttribute.WORKFLOW_NAME]: workflow,
            [OTelAttribute.RUN_AT_TIMESTAMPS]: JSON.stringify([triggerAtIso]),
            [OTelAttribute.ACTION_PAYLOAD]: JSON.stringify(input.input),
            [OTelAttribute.ADDITIONAL_METADATA]: input.additionalMetadata
              ? JSON.stringify(input.additionalMetadata)
              : undefined,
            [OTelAttribute.PRIORITY]: input.priority,
          },
          self.getConfig().excludedAttributes
        );

        return self.tracer.startActiveSpan(
          'hatchet.schedule_workflow',
          {
            kind: SpanKind.PRODUCER,
            attributes,
          },
          (span: Span) => {
            // Inject traceparent into additionalMetadata for context propagation
            const enhancedMetadata: Carrier = { ...(input.additionalMetadata ?? {}) };
            injectContext(enhancedMetadata);

            const enhancedInput = {
              ...input,
              additionalMetadata: enhancedMetadata,
            };

            return original.call(this, workflow, enhancedInput)
              .catch((error: Error) => {
                span.recordException(error);
                span.setStatus({ code: SpanStatusCode.ERROR, message: error?.message });
                throw error;
              })
              .finally(() => {
                span.end();
              });
          }
        );
      };
    });
  }

}

export default HatchetInstrumentor;

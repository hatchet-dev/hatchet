/**
 * Hatchet OpenTelemetry Instrumentor
 *
 * This module provides automatic instrumentation for Hatchet SDK operations
 * including workflow runs, event pushes, and step executions.
 *
 * The instrumentor follows the OpenTelemetry instrumentation pattern,
 * patching module prototypes to automatically instrument all instances.
 */

import type { Context as OtelContext, Span, Attributes } from '@opentelemetry/api';

import type { InstrumentationConfig } from '@opentelemetry/instrumentation';

import { HATCHET_VERSION } from '@hatchet/version';
import { Action } from '@hatchet/clients/dispatcher/action-listener';
import type {
  EventClient,
  PushEventOptions,
  EventWithMetadata,
} from '@hatchet/clients/event/event-client';
import type { AdminClient } from '@hatchet/v1/client/admin';
import type { InternalWorker } from '@hatchet/v1/client/worker/worker-internal';
import type { DurableContext } from '@hatchet/v1/client/worker/context';
import type { ClientConfig } from '@hatchet/clients/hatchet-client/client-config';
import { OTelAttribute, type ActionOTelAttributeValue } from '../util/opentelemetry';
import { parseJSON } from '../util/parse';
import { OpenTelemetryConfig, DEFAULT_CONFIG } from './types';
import { setHatchetSpanAttributes, hatchetSpanAttributes } from './hatchet-span-context';
import type { HatchetBspConfig } from './hatchet-exporter';
import { ScheduledWorkflows } from '../clients/rest/generated/data-contracts';
import { ScheduleClient, CreateScheduledRunInput } from '../v1/client/features/schedules';

try {
  require.resolve('@opentelemetry/api');
  require.resolve('@opentelemetry/instrumentation');
} catch {
  throw new Error(
    'To use HatchetInstrumentor, you must install OpenTelemetry packages: npm install @opentelemetry/api @opentelemetry/instrumentation'
  );
}

/* eslint-disable @typescript-eslint/no-require-imports */
const otelApi = require('@opentelemetry/api') as typeof import('@opentelemetry/api');
const otelInstrumentation =
  require('@opentelemetry/instrumentation') as typeof import('@opentelemetry/instrumentation');
/* eslint-enable @typescript-eslint/no-require-imports */

const { context, propagation, SpanKind, SpanStatusCode, diag } = otelApi;

const {
  InstrumentationBase,
  InstrumentationNodeModuleDefinition,
  InstrumentationNodeModuleFile,
  isWrapped,
} = otelInstrumentation;

type HatchetInstrumentationConfig = OpenTelemetryConfig &
  InstrumentationConfig & {
    /**
     * Enable sending traces to the Hatchet engine's OTLP collector (default: true).
     * Requires @opentelemetry/exporter-trace-otlp-grpc and @opentelemetry/sdk-trace-base.
     * Connection settings (endpoint, token, TLS) are read from the provided clientConfig
     * or from the same environment variables used by the Hatchet client.
     * Set to false to disable.
     */
    enableHatchetCollector?: boolean;

    /**
     * The Hatchet ClientConfig to use for the collector connection.
     * If not provided and enableHatchetCollector is true, config will be loaded
     * from environment variables / .hatchet.yaml.
     */
    clientConfig?: ClientConfig;

    /**
     * Configuration for the BatchSpanProcessor that sends spans to the Hatchet collector.
     */
    bspConfig?: HatchetBspConfig;
  };
type Carrier = Record<string, string>;

const INSTRUMENTOR_NAME = '@hatchet-dev/typescript-sdk';
// FIXME: refactor version check to use the new pattern introduced in #2954
const SUPPORTED_VERSIONS = ['>=1.16.0'];

function extractContext(carrier: Carrier | undefined | null): OtelContext {
  return propagation.extract(context.active(), carrier ?? {});
}

function injectContext(carrier: Carrier): void {
  propagation.inject(context.active(), carrier);
}

function injectSourceInfo(carrier: Carrier): void {
  const store = hatchetSpanAttributes.getStore();
  if (!store) return;

  const wfRunId = store['hatchet.workflow_run_id'];
  const stepRunId = store['hatchet.step_run_id'];

  if (typeof wfRunId === 'string' && typeof stepRunId === 'string') {
    carrier['hatchet__source_workflow_run_id'] = wfRunId;
    carrier['hatchet__source_step_run_id'] = stepRunId;
  }
}

function getActionOtelAttributes(
  action: Action,
  excludedAttributes: string[] = [],
  workerId?: string
): Attributes {
  const attributes = {
    [OTelAttribute.TENANT_ID]: action.tenantId,
    [OTelAttribute.WORKER_ID]: workerId,
    [OTelAttribute.WORKFLOW_RUN_ID]: action.workflowRunId,
    [OTelAttribute.STEP_ID]: action.taskId,
    [OTelAttribute.STEP_RUN_ID]: action.taskRunExternalId,
    [OTelAttribute.RETRY_COUNT]: action.retryCount,
    [OTelAttribute.PARENT_WORKFLOW_RUN_ID]: action.parentWorkflowRunId,
    [OTelAttribute.CHILD_WORKFLOW_INDEX]: action.childWorkflowIndex,
    [OTelAttribute.CHILD_WORKFLOW_KEY]: action.childWorkflowKey,
    [OTelAttribute.ACTION_PAYLOAD]: action.actionPayload,
    [OTelAttribute.WORKFLOW_NAME]: action.jobName,
    [OTelAttribute.ACTION_NAME]: action.actionId,
    [OTelAttribute.STEP_NAME]: action.taskName,
    [OTelAttribute.WORKFLOW_ID]: action.workflowId,
    [OTelAttribute.WORKFLOW_VERSION_ID]: action.workflowVersionId,
  } satisfies Record<ActionOTelAttributeValue, Attributes[string] | undefined>;

  const filtered: Attributes = { instrumentor: 'hatchet' };
  for (const [key, value] of Object.entries(attributes)) {
    if (!excludedAttributes.includes(key) && value !== undefined && value !== '') {
      filtered[key] = value;
    }
  }

  return filtered;
}

function filterAttributes(
  attributes: Record<string, unknown>,
  excludedAttributes: string[] = []
): Attributes {
  const filtered: Attributes = { instrumentor: 'hatchet' };
  for (const [key, value] of Object.entries(attributes)) {
    if (
      !excludedAttributes.includes(key) &&
      value !== undefined &&
      value !== null &&
      value !== '' &&
      value !== '{}' &&
      value !== '[]'
    ) {
      filtered[`hatchet.${key}`] =
        typeof value === 'object' ? JSON.stringify(value) : (value as string | number | boolean);
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
    const mergedConfig = { ...DEFAULT_CONFIG, ...config };
    super(INSTRUMENTOR_NAME, HATCHET_VERSION, mergedConfig);

    if (mergedConfig.enableHatchetCollector) {
      this._setupHatchetCollector(config.clientConfig, config.bspConfig);
    }
  }

  /**
   * Sets up the Hatchet OTLP exporter on the current TracerProvider.
   * Loads client config from environment if not provided.
   */
  private _setupHatchetCollector(clientConfig?: ClientConfig, bspConfig?: HatchetBspConfig): void {
    try {
      /* eslint-disable @typescript-eslint/no-require-imports */
      const { addHatchetExporter } =
        require('./hatchet-exporter') as typeof import('./hatchet-exporter');

      let config = clientConfig;
      if (!config) {
        // Load config from environment (same as HatchetClient would)
        const { ConfigLoader } =
          require('@hatchet/util/config-loader/config-loader') as typeof import('@hatchet/util/config-loader/config-loader');
        config = ConfigLoader.loadClientConfig() as ClientConfig;
      }

      // Get the SDK TracerProvider - either from the global provider or create one
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      let sdkTracerProvider: any;
      try {
        const sdkTrace =
          require('@opentelemetry/sdk-trace-base') as typeof import('@opentelemetry/sdk-trace-base');
        /* eslint-enable @typescript-eslint/no-require-imports */

        // Check if the global tracer provider is an SDK TracerProvider
        const globalProvider = otelApi.trace.getTracerProvider();
        if (globalProvider instanceof sdkTrace.BasicTracerProvider) {
          sdkTracerProvider = globalProvider;
        } else {
          // Create a new SDK TracerProvider and set it as global
          sdkTracerProvider = new sdkTrace.BasicTracerProvider();
          sdkTracerProvider.register();
        }
      } catch {
        diag.warn(
          'hatchet instrumentation: @opentelemetry/sdk-trace-base is required for enableHatchetCollector'
        );
        return;
      }

      addHatchetExporter(sdkTracerProvider, config, bspConfig);
      diag.info('hatchet instrumentation: Hatchet OTLP collector enabled');
    } catch (e) {
      diag.warn(`hatchet instrumentation: Failed to set up Hatchet collector: ${e}`);
    }
  }

  override setConfig(config: Partial<HatchetInstrumentationConfig> = {}): void {
    super.setConfig({ ...DEFAULT_CONFIG, ...config });
  }

  private readonly _getConfig = (): HatchetInstrumentationConfig => this.getConfig();

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

    const durableContextModuleFile = new InstrumentationNodeModuleFile(
      '@hatchet-dev/typescript-sdk/v1/client/worker/context.js',
      SUPPORTED_VERSIONS,
      this.patchDurableContext.bind(this),
      this.unpatchDurableContext.bind(this)
    );

    const moduleDefinition = new InstrumentationNodeModuleDefinition(
      INSTRUMENTOR_NAME,
      SUPPORTED_VERSIONS,
      undefined,
      undefined,
      [
        eventClientModuleFile,
        adminClientModuleFile,
        workerModuleFile,
        scheduleClientModuleFile,
        durableContextModuleFile,
      ]
    );

    return [moduleDefinition];
  }

  private patchEventClient(moduleExports: unknown, _moduleVersion?: string): unknown {
    const exports = moduleExports as { EventClient?: { prototype: EventClient } };
    if (!exports?.EventClient?.prototype) {
      diag.debug('hatchet instrumentation: EventClient not found in module exports');
      return moduleExports;
    }
    this._patchPushEvent(exports.EventClient.prototype);
    this._patchBulkPushEvent(exports.EventClient.prototype);

    return moduleExports;
  }

  private unpatchEventClient(moduleExports: unknown, _moduleVersion?: string): unknown {
    const exports = moduleExports as { EventClient?: { prototype: EventClient } };
    if (!exports?.EventClient?.prototype) {
      return moduleExports;
    }

    if (isWrapped(exports.EventClient.prototype.push)) {
      this._unwrap(exports.EventClient.prototype, 'push');
    }
    if (isWrapped(exports.EventClient.prototype.bulkPush)) {
      this._unwrap(exports.EventClient.prototype, 'bulkPush');
    }

    return moduleExports;
  }

  private _patchPushEvent(prototype: EventClient): void {
    if (isWrapped(prototype.push)) {
      this._unwrap(prototype, 'push');
    }
    const { tracer, _getConfig: getConfig } = this;

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
            [OTelAttribute.FILTER_SCOPE]: options.scope,
          },
          getConfig().excludedAttributes
        );

        return tracer.startActiveSpan(
          'hatchet.push_event',
          {
            kind: SpanKind.PRODUCER,
            attributes,
          },
          (span: Span) => {
            const enhancedMetadata: Carrier = { ...(options.additionalMetadata ?? {}) };
            injectContext(enhancedMetadata);
            injectSourceInfo(enhancedMetadata);

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
    const { tracer, _getConfig: getConfig } = this;

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
          getConfig().excludedAttributes
        );

        return tracer.startActiveSpan(
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
              injectSourceInfo(enhancedMetadata);
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

  private patchAdminClient(moduleExports: unknown, _moduleVersion?: string): unknown {
    const exports = moduleExports as { AdminClient?: { prototype: AdminClient } };
    if (!exports?.AdminClient?.prototype) {
      diag.debug('hatchet instrumentation: AdminClient not found in module exports');
      return moduleExports;
    }

    this._patchRunWorkflow(exports.AdminClient.prototype);
    this._patchRunWorkflows(exports.AdminClient.prototype);

    return moduleExports;
  }

  private unpatchAdminClient(moduleExports: unknown, _moduleVersion?: string): unknown {
    const exports = moduleExports as { AdminClient?: { prototype: AdminClient } };
    if (!exports?.AdminClient?.prototype) {
      return moduleExports;
    }

    if (isWrapped(exports.AdminClient.prototype.runWorkflow)) {
      this._unwrap(exports.AdminClient.prototype, 'runWorkflow');
    }
    if (isWrapped(exports.AdminClient.prototype.runWorkflows)) {
      this._unwrap(exports.AdminClient.prototype, 'runWorkflows');
    }

    return moduleExports;
  }

  private _patchRunWorkflow(prototype: AdminClient): void {
    if (isWrapped(prototype.runWorkflow)) {
      this._unwrap(prototype, 'runWorkflow');
    }
    const { tracer, _getConfig: getConfig } = this;

    this._wrap(prototype, 'runWorkflow', (original: AdminClient['runWorkflow']) => {
      return async function wrappedRunWorkflow(
        this: AdminClient,
        workflowName: string,
        input: unknown,
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
          getConfig().excludedAttributes
        );

        return tracer.startActiveSpan(
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

            return original
              .call(this, workflowName, input, enhancedOptions)
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
    const { tracer, _getConfig: getConfig } = this;

    this._wrap(prototype, 'runWorkflows', (original: AdminClient['runWorkflows']) => {
      return async function wrappedRunWorkflows(
        this: AdminClient,
        workflowRuns: Array<{
          workflowName: string;
          input: unknown;
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
          getConfig().excludedAttributes
        );

        return tracer.startActiveSpan(
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

            return original
              .call(this, enhancedWorkflowRuns, batchSize)
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

  private patchWorker(moduleExports: unknown, _moduleVersion?: string): unknown {
    const exports = moduleExports as { InternalWorker?: { prototype: InternalWorker } };
    if (!exports?.InternalWorker?.prototype) {
      diag.debug('hatchet instrumentation: InternalWorker not found in module exports');
      return moduleExports;
    }

    this._patchHandleStartStepRun(exports.InternalWorker.prototype);
    this._patchHandleCancelStepRun(exports.InternalWorker.prototype);

    return moduleExports;
  }

  private unpatchWorker(moduleExports: unknown, _moduleVersion?: string): unknown {
    const exports = moduleExports as { InternalWorker?: { prototype: InternalWorker } };
    if (!exports?.InternalWorker?.prototype) {
      return moduleExports;
    }

    if (isWrapped(exports.InternalWorker.prototype.handleStartStepRun)) {
      this._unwrap(exports.InternalWorker.prototype, 'handleStartStepRun');
    }
    if (isWrapped(exports.InternalWorker.prototype.handleCancelStepRun)) {
      this._unwrap(exports.InternalWorker.prototype, 'handleCancelStepRun');
    }

    return moduleExports;
  }

  // IMPORTANT: Keep this wrapper's signature in sync with InternalWorker.handleStartStepRun
  private _patchHandleStartStepRun(prototype: InternalWorker): void {
    if (isWrapped(prototype.handleStartStepRun)) {
      this._unwrap(prototype, 'handleStartStepRun');
    }
    const { tracer, _getConfig: getConfig } = this;

    this._wrap(
      prototype,
      'handleStartStepRun',
      (original: InternalWorker['handleStartStepRun']) => {
        return async function wrappedHandleStartStepRun(
          this: InternalWorker,
          action: Action
        ): Promise<Error | undefined> {
          const additionalMetadata = action.additionalMetadata
            ? parseJSON(action.additionalMetadata)
            : undefined;
          const parentContext = extractContext(additionalMetadata);
          const attributes = getActionOtelAttributes(
            action,
            getConfig().excludedAttributes,
            this.workerId
          );

          let spanName = 'hatchet.start_step_run';
          if (getConfig().includeTaskNameInSpanName) {
            spanName += `.${action.actionId}`;
          }

          // Store hatchet.* attributes in async context so the SpanProcessor
          // can inject them into child spans (mirrors Go/Python attribute propagation).
          const hatchetAttrs: Attributes = {};
          for (const [key, value] of Object.entries(attributes)) {
            if (value !== undefined) {
              hatchetAttrs[`hatchet.${key}`] = value;
            }
          }
          setHatchetSpanAttributes(hatchetAttrs);

          return tracer.startActiveSpan(
            spanName,
            {
              kind: SpanKind.CONSUMER,
              attributes,
            },
            parentContext,
            (span: Span) => {
              return original
                .call(this, action)
                .then((taskError: Error | undefined) => {
                  if (taskError instanceof Error) {
                    span.recordException(taskError);
                    span.setStatus({ code: SpanStatusCode.ERROR, message: taskError.message });
                  } else {
                    span.setStatus({ code: SpanStatusCode.OK });
                  }
                  return taskError;
                })
                .finally(() => {
                  span.end();
                });
            }
          );
        };
      }
    );
  }

  private _patchHandleCancelStepRun(prototype: InternalWorker): void {
    if (isWrapped(prototype.handleCancelStepRun)) {
      this._unwrap(prototype, 'handleCancelStepRun');
    }
    const { tracer } = this;

    this._wrap(
      prototype,
      'handleCancelStepRun',
      (original: InternalWorker['handleCancelStepRun']) => {
        return async function wrappedHandleCancelStepRun(
          this: InternalWorker,
          action: Action
        ): Promise<void> {
          const attributes: Attributes = {
            instrumentor: 'hatchet',
            [`hatchet.${OTelAttribute.STEP_RUN_ID}`]: action.taskRunExternalId,
          };

          return tracer.startActiveSpan(
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
      }
    );
  }

  private patchScheduleClient(moduleExports: unknown, _moduleVersion?: string): unknown {
    const exports = moduleExports as { ScheduleClient?: { prototype: ScheduleClient } };
    if (!exports?.ScheduleClient?.prototype) {
      diag.debug('hatchet instrumentation: ScheduleClient not found in module exports');
      return moduleExports;
    }

    this._patchScheduleCreate(exports.ScheduleClient.prototype);

    return moduleExports;
  }

  private unpatchScheduleClient(moduleExports: unknown, _moduleVersion?: string): unknown {
    const exports = moduleExports as { ScheduleClient?: { prototype: ScheduleClient } };
    if (!exports?.ScheduleClient?.prototype) {
      return moduleExports;
    }

    if (isWrapped(exports.ScheduleClient.prototype.create)) {
      this._unwrap(exports.ScheduleClient.prototype, 'create');
    }

    return moduleExports;
  }

  // IMPORTANT: Keep this wrapper's signature in sync with ScheduleClient.create
  private _patchScheduleCreate(prototype: ScheduleClient): void {
    if (isWrapped(prototype.create)) {
      this._unwrap(prototype, 'create');
    }
    const { tracer, _getConfig: getConfig } = this;

    this._wrap(prototype, 'create', (original: ScheduleClient['create']) => {
      return async function wrappedCreate(
        this: ScheduleClient,
        workflow: string,
        input: CreateScheduledRunInput
      ): Promise<ScheduledWorkflows> {
        const triggerAtIso =
          input.triggerAt instanceof Date
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
          getConfig().excludedAttributes
        );

        return tracer.startActiveSpan(
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

            return original
              .call(this, workflow, enhancedInput as CreateScheduledRunInput)
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
  // --- DurableContext patching ---

  private patchDurableContext(moduleExports: unknown): unknown {
    const exports = moduleExports as { DurableContext?: { prototype: DurableContext<unknown> } };
    if (!exports?.DurableContext?.prototype) {
      return moduleExports;
    }

    this._patchWaitFor(exports.DurableContext.prototype);
    return moduleExports;
  }

  private unpatchDurableContext(moduleExports: unknown): void {
    const exports = moduleExports as { DurableContext?: { prototype: DurableContext<unknown> } };
    if (!exports?.DurableContext?.prototype) {
      return;
    }
    if (isWrapped(exports.DurableContext.prototype.waitFor)) {
      this._unwrap(exports.DurableContext.prototype, 'waitFor');
    }
  }

  // IMPORTANT: Keep this wrapper's signature in sync with DurableContext.waitFor
  private _patchWaitFor(prototype: DurableContext<unknown>): void {
    if (isWrapped(prototype.waitFor)) {
      this._unwrap(prototype, 'waitFor');
    }

    const { tracer } = this;

    this._wrap(prototype, 'waitFor', (original: DurableContext<unknown>['waitFor']) => {
      return async function wrappedWaitFor(
        this: DurableContext<unknown>,
        ...args: Parameters<DurableContext<unknown>['waitFor']>
      ): Promise<Record<string, unknown>> {
        return tracer.startActiveSpan(
          'hatchet.durable.wait_for',
          {
            kind: SpanKind.INTERNAL,
            attributes: {
              instrumentor: 'hatchet',
              'hatchet.step_run_id': this.action?.taskRunExternalId ?? '',
            },
          },
          async (span: Span) => {
            try {
              const result = await original.apply(this, args);
              span.setStatus({ code: SpanStatusCode.OK });
              return result;
            } catch (error: unknown) {
              if (error instanceof Error) {
                span.recordException(error);
                span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
              }
              throw error;
            } finally {
              span.end();
            }
          }
        );
      };
    });
  }
}

export default HatchetInstrumentor;

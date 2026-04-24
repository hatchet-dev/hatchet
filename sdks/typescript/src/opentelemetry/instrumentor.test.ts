import { propagation } from '@opentelemetry/api';
import { W3CTraceContextPropagator } from '@opentelemetry/core';
import {
  BasicTracerProvider,
  InMemorySpanExporter,
  SimpleSpanProcessor,
} from '@opentelemetry/sdk-trace-base';
import { HatchetInstrumentor } from './instrumentor';

describe('HatchetInstrumentor worker spans', () => {
  it('creates a workflow-level parent span for dashboard-triggered step runs', async () => {
    const exporter = new InMemorySpanExporter();
    const provider = new BasicTracerProvider({
      spanProcessors: [new SimpleSpanProcessor(exporter)],
    });

    propagation.setGlobalPropagator(new W3CTraceContextPropagator());

    const dashboardTracer = provider.getTracer('dashboard-test');
    const dashboardSpan = dashboardTracer.startSpan('dashboard-trigger');
    const dashboardSpanContext = dashboardSpan.spanContext();
    const carrier: Record<string, string> = {
      traceparent: `00-${dashboardSpanContext.traceId}-${dashboardSpanContext.spanId}-01`,
    };
    dashboardSpan.end();

    class FakeWorker {
      workerId = 'worker-1';

      async handleStartStepRun(_action: unknown) {
        return undefined;
      }

      async handleCancelStepRun() {
        return undefined;
      }
    }

    const instrumentor = new HatchetInstrumentor();
    instrumentor.setTracerProvider(provider);
    (instrumentor as any)._patchHandleStartStepRun(FakeWorker.prototype);

    const worker = new FakeWorker();
    await worker.handleStartStepRun({
      tenantId: 'tenant-1',
      workflowRunId: 'workflow-run-1',
      taskId: 'task-1',
      taskRunExternalId: 'step-run-1',
      retryCount: 0,
      parentWorkflowRunId: '',
      childWorkflowIndex: 0,
      childWorkflowKey: '',
      actionPayload: '{"input":true}',
      jobName: 'find-subprocessors',
      actionId: 'find-subprocessors:resolve-parent-company',
      taskName: 'resolve-parent-company',
      workflowId: 'workflow-1',
      workflowVersionId: 'workflow-version-1',
      additionalMetadata: JSON.stringify(carrier),
    } as any);

    const spans = exporter.getFinishedSpans();
    const workflowSpan = spans.find((span) => span.name === 'hatchet.workflow_run');
    const stepSpan = spans.find((span) => span.name === 'hatchet.start_step_run');

    expect(workflowSpan).toBeDefined();
    expect(stepSpan).toBeDefined();
    expect(stepSpan?.parentSpanContext?.spanId).toBe(workflowSpan?.spanContext().spanId);
    expect(workflowSpan?.parentSpanContext?.spanId).toBe(dashboardSpanContext.spanId);
  });
});

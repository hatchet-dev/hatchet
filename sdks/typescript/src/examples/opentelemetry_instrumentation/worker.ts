import { hatchet } from './client';
import { getTracer } from './tracer';
import { SpanStatusCode } from '@opentelemetry/api';

const tracer = getTracer('opentelemetry-worker');

export const otelWorkflow = hatchet.workflow({
  name: 'otelworkflowtypescript',
});

otelWorkflow.task({
  name: 'step-with-custom-spans',
  fn: async (input, ctx) => {
    return tracer.startActiveSpan('custom-business-logic', async (span) => {
      try {
        console.log('Executing step with custom tracing...');
        await new Promise((resolve) => setTimeout(resolve, 100));

        return { result: 'success' };
      } finally {
        span.end();
      }
    });
  },
});

/**
 * Task demonstrating that span hierarchy is preserved even when errors occur.
 */
otelWorkflow.task({
  name: 'step-with-error',
  fn: async (input, ctx) => {
    return tracer.startActiveSpan('custom-span-with-error', async (span) => {
      try {
        throw new Error('Intentional error for demonstration');
      } catch (error: any) {
        span.recordException(error);
        span.setStatus({ code: SpanStatusCode.ERROR, message: error?.message });
        throw error;
      } finally {
        span.end();
      }
    });
  },
});

/**
 * Task that is automatically instrumented without any manual span creation.
 */
otelWorkflow.task({
  name: 'auto-instrumented-step',
  fn: async (input, ctx) => {
    console.log('This step is automatically traced without manual span code');
    return { automatically: 'instrumented' };
  },
});

/**
 * Task demonstrating error handling in auto-instrumented steps.
 */
otelWorkflow.task({
  name: 'auto-instrumented-step-with-error',
  fn: async (input, ctx) => {
    throw new Error('Auto-instrumented step error');
  },
});

async function main() {
  console.log('Starting OpenTelemetry instrumented worker...');
  console.log('Instrumentation is automatic via module patching.');

  const worker = await hatchet.worker('otel-example-worker-ts', {
    slots: 1,
    workflows: [otelWorkflow],
  });

  await worker.start();
}

if (require.main === module) {
  main().catch(console.error);
}

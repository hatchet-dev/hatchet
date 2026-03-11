import { hatchet } from '../hatchet-client';
import { getTracer } from './tracer';
import { SpanStatusCode } from '@opentelemetry/api';

const tracer = getTracer('opentelemetry-worker');

export const otelWorkflow = hatchet.workflow({
  name: 'otelworkflowtypescript',
});

otelWorkflow.task({
  name: 'step-with-custom-spans',
  fn: async () => {
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

otelWorkflow.task({
  name: 'step-with-error',
  fn: async () => {
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

otelWorkflow.task({
  name: 'auto-instrumented-step',
  fn: async () => {
    console.log('This step is automatically traced without manual span code');
    return { automatically: 'instrumented' };
  },
});

otelWorkflow.task({
  name: 'auto-instrumented-step-with-error',
  fn: async () => {
    throw new Error('Auto-instrumented step error');
  },
});

async function main() {
  const worker = await hatchet.worker('otel-example-worker-ts', {
    slots: 1,
    workflows: [otelWorkflow],
  });

  await worker.start();
}

if (require.main === module) {
  main().catch(console.error);
}

import { getTracer } from './tracer';

import { SpanStatusCode, type Span } from '@opentelemetry/api';
import { hatchet } from './client';
import { otelWorkflow } from './worker';

const tracer = getTracer('opentelemetry-triggers');

const ADDITIONAL_METADATA = { source: 'otel-example', version: '1.0' };

async function pushEvent() {
  console.log('\n--- Push Event ---');

  return tracer.startActiveSpan('push_event', async (span: Span) => {
    try {
      await hatchet.events.push(
        'otel:event',
        { message: 'Hello from instrumented trigger!' },
        { additionalMetadata: ADDITIONAL_METADATA }
      );
      console.log('Event pushed successfully');
    } finally {
      span.end();
    }
  });
}

async function bulkPushEvents() {
  console.log('\n--- Bulk Push Events ---');

  return tracer.startActiveSpan('bulk_push_event', async (span: Span) => {
    try {
      await hatchet.events.bulkPush('otel:event', [
        {
          payload: { message: 'Bulk event 1' },
          additionalMetadata: ADDITIONAL_METADATA,
        },
        {
          payload: { message: 'Bulk event 2' },
          additionalMetadata: ADDITIONAL_METADATA,
        },
        {
          payload: { message: 'Bulk event 3' },
          additionalMetadata: ADDITIONAL_METADATA,
        },
      ]);
      console.log('Bulk events pushed successfully');
    } finally {
      span.end();
    }
  });
}

async function runWorkflow() {
  console.log('\n--- Run Workflow ---');

  return tracer.startActiveSpan('run_workflow', async (span: Span) => {
    try {
      const workflowRun = await hatchet.admin.runWorkflow(otelWorkflow.name, {}, {
        additionalMetadata: ADDITIONAL_METADATA,
      });
      const runId = await workflowRun.getWorkflowRunId();
      console.log(`Started workflow run: ${runId}`);

      const result = await workflowRun.output;
      span.setStatus({ code: SpanStatusCode.OK, message: 'Workflow completed' });
      console.log(`Workflow completed with result:`, result);
    } catch (error: any) {
      const errorMessage = Array.isArray(error) ? error.join(', ') : error?.message || String(error);
      console.error('Workflow failed:', errorMessage);
      span.recordException(error);
      span.setStatus({ code: SpanStatusCode.ERROR, message: errorMessage });
    } finally {
      span.end();
    }
  });
}

async function runWorkflows() {
  console.log('\n--- Run Workflows (Bulk) ---');

  return tracer.startActiveSpan('run_workflows', async (span: Span) => {
    try {
      const refs = await hatchet.admin.runWorkflows([
        {
          workflowName: otelWorkflow.name,
          input: {},
          options: { additionalMetadata: ADDITIONAL_METADATA },
        },
        {
          workflowName: otelWorkflow.name,
          input: {},
          options: { additionalMetadata: ADDITIONAL_METADATA },
        },
      ]);
      console.log(`Started ${refs.length} workflow runs`);

      const results = await Promise.all(refs.map((ref: { result: () => Promise<unknown> }) => ref.result()));
      span.setStatus({ code: SpanStatusCode.OK, message: 'Workflows completed' });
      console.log(`Workflows completed with results:`, results);
    } catch (error: any) {
      const errorMessage = Array.isArray(error) ? error.join(', ') : error?.message || String(error);
      console.error('Workflows failed:', errorMessage);
      span.recordException(error);
      span.setStatus({ code: SpanStatusCode.ERROR, message: errorMessage });
    } finally {
      span.end();
    }
  });
}


async function main() {
  console.log('OpenTelemetry Triggers Example');
  console.log('==============================\n');

  await pushEvent();
  await bulkPushEvents();
  await runWorkflow();
  await runWorkflows();

  console.log('\n--- Waiting for spans to be exported... ---');
  await new Promise((resolve) => setTimeout(resolve, 5000));
  console.log('Done!');

  process.exit(0);
}

if (require.main === module) {
  main().catch(console.error);
}

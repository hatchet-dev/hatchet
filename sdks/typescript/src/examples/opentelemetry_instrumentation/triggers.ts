import { hatchet } from './client';
import { getTracer } from './tracer';

const tracer = getTracer('opentelemetry-triggers');

const WORKFLOW_NAME = 'otelworkflowtypescript';
const ADDITIONAL_METADATA = { source: 'otel-example', version: '1.0' };

async function pushEvent() {
  console.log('\n--- Push Event ---');

  return tracer.startActiveSpan('push_event', async (span) => {
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

  return tracer.startActiveSpan('bulk_push_events', async (span) => {
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

  return tracer.startActiveSpan('run_workflow', async (span) => {
    try {
      const workflowRun = await hatchet.admin.runWorkflow(WORKFLOW_NAME, {}, {
        additionalMetadata: ADDITIONAL_METADATA,
      });
      const runId = await workflowRun.getWorkflowRunId();
      console.log(`Started workflow run: ${runId}`);

      // Optionally wait for result
      const result = await workflowRun.result();
      console.log(`Workflow completed with result:`, result);
    } finally {
      span.end();
    }
  });
}

async function runWorkflows() {
  console.log('\n--- Run Workflows (Bulk) ---');

  return tracer.startActiveSpan('run_workflows', async (span) => {
    try {
      const refs = await hatchet.admin.runWorkflows([
        {
          workflowName: WORKFLOW_NAME,
          input: {},
          options: { additionalMetadata: ADDITIONAL_METADATA },
        },
        {
          workflowName: WORKFLOW_NAME,
          input: {},
          options: { additionalMetadata: ADDITIONAL_METADATA },
        },
      ]);
      console.log(`Started ${refs.length} workflow runs`);

      // Optionally wait for results
      const results = await Promise.all(refs.map((ref) => ref.result()));
      console.log(`Workflows completed with results:`, results);
    } finally {
      span.end();
    }
  });
}

async function triggerWithoutParentSpan() {
  console.log('\n--- Trigger Without Parent Span ---');

  await hatchet.events.push('otel:standalone', { standalone: true });
  console.log('Standalone event pushed (auto-instrumented)');
}

async function main() {
  console.log('OpenTelemetry Triggers Example');
  console.log('==============================\n');

  await pushEvent();
  await bulkPushEvents();
  await runWorkflow();
  await runWorkflows();
  await triggerWithoutParentSpan();

  console.log('\n--- Waiting for spans to be exported... ---');
  await new Promise((resolve) => setTimeout(resolve, 5000));
  console.log('Done!');

  process.exit(0);
}

if (require.main === module) {
  main().catch(console.error);
}

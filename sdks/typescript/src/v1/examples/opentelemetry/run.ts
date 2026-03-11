import { getTracer } from './tracer';

import { SpanStatusCode, type Span } from '@opentelemetry/api';
import { hatchet } from '../hatchet-client';
import { otelWorkflow } from './worker';

const tracer = getTracer('opentelemetry-triggers');

async function runWorkflow() {
  return tracer.startActiveSpan('run_workflow', async (span: Span) => {
    try {
      const workflowRun = await hatchet.admin.runWorkflow(otelWorkflow.name, {});
      const runId = await workflowRun.getWorkflowRunId();
      console.log(`Started workflow run: ${runId}`);

      const result = await workflowRun.output;
      span.setStatus({ code: SpanStatusCode.OK, message: 'Workflow completed' });
      console.log(`Workflow completed with result:`, result);
    } catch (error: any) {
      const errorMessage = Array.isArray(error)
        ? error.join(', ')
        : error?.message || String(error);
      console.error('Workflow failed:', errorMessage);
      span.recordException(error);
      span.setStatus({ code: SpanStatusCode.ERROR, message: errorMessage });
    } finally {
      span.end();
    }
  });
}

async function main() {
  await runWorkflow();

  console.log('Waiting for spans to be exported...');
  await new Promise((resolve) => setTimeout(resolve, 5000));

  process.exit(0);
}

if (require.main === module) {
  main().catch(console.error);
}

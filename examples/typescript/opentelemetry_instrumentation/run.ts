import { SpanStatusCode, type Span } from '@opentelemetry/api';
import { hatchet } from '../hatchet-client';
import { initOtel, getTracer } from './setup';
import { orderWorkflow } from './worker';

initOtel();

const tracer = getTracer('otel-instrumentation-triggers');

// > Trigger
async function runWorkflow() {
  return tracer.startActiveSpan('trigger_order_processing', async (span: Span) => {
    try {
      const workflowRun = await hatchet.admin.runWorkflow(orderWorkflow.name, {
        orderId: 'order-123',
        customerId: 'cust-456',
        amount: 4999,
      });
      const runId = await workflowRun.getWorkflowRunId();
      console.log(`Started workflow run: ${runId}`);

      const result = await workflowRun.output;
      span.setStatus({ code: SpanStatusCode.OK, message: 'Order processed' });
      console.log('Order completed:', result);
    } catch (error: any) {
      const errorMessage = Array.isArray(error)
        ? error.join(', ')
        : error?.message || String(error);
      console.error('Order processing failed:', errorMessage);
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

import { hatchet } from '../hatchet-client';
import { initOtel, getTracer } from './setup';
import { SpanStatusCode } from '@opentelemetry/api';

initOtel();

const tracer = getTracer('otel-instrumentation-worker');

type OrderInput = {
  orderId: string;
  customerId: string;
  amount: number;
};

export const orderWorkflow = hatchet.workflow<OrderInput>({
  name: 'otel-order-processing-ts',
});

// > Custom Spans
const validateOrder = orderWorkflow.task({
  name: 'validate-order',
  fn: async (input) => {
    await tracer.startActiveSpan('order.validate.schema', async (span) => {
      await new Promise((resolve) => setTimeout(resolve, 10));
      span.setAttribute('order.id', input.orderId);
      span.end();
    });

    await tracer.startActiveSpan('order.validate.fraud-check', async (span) => {
      await new Promise((resolve) => setTimeout(resolve, 20));
      span.setAttribute('fraud.score', 0.05);
      span.setAttribute('fraud.decision', 'allow');
      span.end();
    });

    return { valid: true, orderId: input.orderId };
  },
});

const chargePayment = orderWorkflow.task({
  name: 'charge-payment',
  parents: [validateOrder],
  fn: async (input, ctx) => {
    const validated = await ctx.parentOutput(validateOrder);

    return tracer.startActiveSpan('payment.process', async (paySpan) => {
      try {
        await tracer.startActiveSpan('payment.tokenize-card', async (span) => {
          await new Promise((resolve) => setTimeout(resolve, 15));
          span.setAttribute('payment.provider', 'stripe');
          span.end();
        });

        await tracer.startActiveSpan('payment.charge', async (span) => {
          await new Promise((resolve) => setTimeout(resolve, 30));
          span.setAttribute('payment.amount_cents', input.amount);
          span.setAttribute('payment.currency', 'USD');
          span.end();
        });

        paySpan.setStatus({ code: SpanStatusCode.OK });
        return {
          transactionId: `txn-${validated.orderId}`,
          charged: input.amount,
        };
      } finally {
        paySpan.end();
      }
    });
  },
});

const reserveInventory = orderWorkflow.task({
  name: 'reserve-inventory',
  parents: [validateOrder],
  fn: async (input) => {
    await tracer.startActiveSpan('inventory.check-availability', async (span) => {
      await new Promise((resolve) => setTimeout(resolve, 10));
      span.setAttribute('inventory.sku_count', 3);
      span.setAttribute('inventory.all_available', true);
      span.end();
    });

    await tracer.startActiveSpan('inventory.reserve', async (span) => {
      await new Promise((resolve) => setTimeout(resolve, 15));
      span.setAttribute('inventory.warehouse', 'us-east-1');
      span.end();
    });

    return {
      reservationId: `res-${input.orderId}`,
      itemsReserved: 3,
    };
  },
});

orderWorkflow.task({
  name: 'send-confirmation',
  parents: [chargePayment, reserveInventory],
  fn: async (input, ctx) => {
    const payment = await ctx.parentOutput(chargePayment);
    const inventory = await ctx.parentOutput(reserveInventory);

    await tracer.startActiveSpan('notification.render-template', async (span) => {
      await new Promise((resolve) => setTimeout(resolve, 5));
      span.setAttribute('template.name', 'order-confirmation');
      span.end();
    });

    await tracer.startActiveSpan('notification.send-email', async (span) => {
      await new Promise((resolve) => setTimeout(resolve, 20));
      span.setAttribute('email.to', 'customer@example.com');
      span.setAttribute('email.provider', 'sendgrid');
      span.end();
    });

    return {
      sent: true,
      transactionId: payment.transactionId,
      reservationId: inventory.reservationId,
    };
  },
});

// > Worker
async function main() {
  const worker = await hatchet.worker('otel-instrumentation-worker-ts', {
    workflows: [orderWorkflow],
  });

  await worker.start();
}

if (require.main === module) {
  main().catch(console.error);
}

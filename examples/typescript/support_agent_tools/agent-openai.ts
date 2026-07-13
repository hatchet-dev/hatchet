import {
  createLookupCustomerToolOpenai,
  createCheckOrderStatusToolOpenai,
  createTicketToolOpenai,
} from './tools';

async function main() {
  const lookupCustomerTool = createLookupCustomerToolOpenai();
  const checkOrderStatusTool = createCheckOrderStatusToolOpenai();
  const ticketTool = createTicketToolOpenai();

  // Dynamic import — @openai/agents crashes at load time with Zod v3, so we delay
  // the import until after mcpTool has already verified Zod v4 is available.
  const { Agent, run } = await import('@openai/agents');

  const agent = new Agent({
    name: 'support-agent',
    tools: [lookupCustomerTool, checkOrderStatusTool, ticketTool],
  });

  const result = await run(
    agent,
    'Customer C-100 says order ORD-9987 has not arrived. ' +
      'Look up the customer, check the order status, and create a ' +
      'support ticket if the order has a known issue or delayed delivery. ' +
      'If you create a ticket, use priority "high", subject ' +
      '"Delayed order ORD-9987", and a body that summarizes the known ' +
      'carrier delay. Then summarize what happened.'
  );
  console.log(result.finalOutput);
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => {
      process.exit(0);
    });
}

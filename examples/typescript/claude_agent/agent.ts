import { createLookupCustomerTool, createCheckOrderStatusTool, createTicketTool } from './tools';

async function main() {
  const lookupCustomerTool = createLookupCustomerTool();
  const checkOrderStatusTool = createCheckOrderStatusTool();
  const ticketTool = createTicketTool();

  // The Claude Agent SDK is ESM-only, so avoid loading it at module import time.
  // Run this example with an ESM-compatible TypeScript runner.
  const { query, createSdkMcpServer } = await import('@anthropic-ai/claude-agent-sdk');

  // Wrap the tools in an in-process MCP server
  const supportServer = createSdkMcpServer({
    name: 'support',
    version: '1.0.0',
    tools: [lookupCustomerTool, checkOrderStatusTool, ticketTool],
  });

  for await (const message of query({
    prompt:
      'Customer C-100 says order ORD-9987 has not arrived. ' +
      'Look up the customer, check the order status, and create a ' +
      'support ticket if the order has a known issue or delayed delivery. ' +
      'If you create a ticket, use priority "high", subject ' +
      '"Delayed order ORD-9987", and a body that summarizes the known ' +
      'carrier delay. Then summarize what happened.',
    options: {
      mcpServers: { support: supportServer },
      allowedTools: [
        `mcp__${supportServer.name}__${lookupCustomerTool.name}`,
        `mcp__${supportServer.name}__${checkOrderStatusTool.name}`,
        `mcp__${supportServer.name}__${ticketTool.name}`,
      ],
    },
  })) {
    // "result" is the final message after all tool calls complete
    if (message.type === 'result' && message.subtype === 'success') {
      console.log(message.result);
    }
  }
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => {
      process.exit(0);
    });
}

// eslint-disable-next-line
import { createTemperatureWorkflowToolClaude } from './workflow';

async function main() {
  const temperatureTool = createTemperatureWorkflowToolClaude();

  // The Claude Agent SDK is ESM-only, so avoid loading it at module import time.
  // Run this example with an ESM-compatible TypeScript runner.
  const { query, createSdkMcpServer } = await import('@anthropic-ai/claude-agent-sdk');

  // Wrap the tool in an in-process MCP server
  const weatherServer = createSdkMcpServer({
    name: 'weather',
    version: '1.0.0',
    tools: [temperatureTool],
  });

  for await (const message of query({
    prompt: "What's the temperature in San Francisco?",
    options: {
      mcpServers: { weather: weatherServer },
      allowedTools: [`mcp__${weatherServer.name}__${temperatureTool.name}`],
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

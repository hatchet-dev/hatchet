import { createTemperatureWorkflowToolClaude } from './workflow';

async function main() {
  const temperatureTool = createTemperatureWorkflowToolClaude();

  // Dynamic import — the Claude Agent SDK is ESM-only, so we use import()
  // instead of a static import to avoid ERR_REQUIRE_ESM under ts-node/CJS.
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

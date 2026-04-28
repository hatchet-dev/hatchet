import { getTemperature, getTemperatureWorkflow } from './workflow';
import { query } from '@anthropic-ai/claude-agent-sdk';
import { createSdkMcpServer } from '@anthropic-ai/claude-agent-sdk';

async function main() {
  // Generate a tool from a standalone task
  // const temperatureTool = getTemperature.mcpTool('claude');

  // Or from a workflow
  const temperatureTool = getTemperatureWorkflow.mcpTool('claude');

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

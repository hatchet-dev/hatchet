import { getTemperature } from './workflow';
import { query } from '@anthropic-ai/claude-agent-sdk';
import { createSdkMcpServer, tool } from '@anthropic-ai/claude-agent-sdk';

async function main() {
  // Define a tool: name, description, input schema, handler
  const temperatureTool = getTemperature.mcpTool('Get the current temperature at a location');

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
      allowedTools: ['mcp__weather__getTemperature'],
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

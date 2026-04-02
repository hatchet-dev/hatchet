// eslint-disable-next-line
import { getTemperature, getTemperatureWorkflow } from './workflow';
import { Agent, run } from '@openai/agents';

async function main() {
  // Generate a tool from a standalone task
  // const temperatureTool = getTemperature.mcpTool('openai');

  // Or from a workflow
  const temperatureTool = getTemperatureWorkflow.mcpTool('openai');

  const agent = new Agent({
    name: 'Agent',
    tools: [temperatureTool],
  });
  const result = await run(agent, 'What is the weather in San Francisco?');
  console.log(result.finalOutput);
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => {
      process.exit(0);
    });
}

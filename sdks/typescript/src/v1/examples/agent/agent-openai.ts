// eslint-disable-next-line
import { getTemperature, getTemperatureWorkflow } from './workflow';

async function main() {
  // Generate a tool from a standalone task
  // const temperatureTool = getTemperature.mcpTool('openai');

  // Or from a workflow.
  // mcpTool validates Zod v4 is installed before loading @openai/agents, so call it first
  // to get a clear error message if the wrong Zod version is present.
  const temperatureTool = getTemperatureWorkflow.mcpTool('openai');

  // Dynamic import — @openai/agents crashes at load time with Zod v3, so we delay
  // the import until after mcpTool has already verified Zod v4 is available.
  const { Agent, run } = await import('@openai/agents');

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

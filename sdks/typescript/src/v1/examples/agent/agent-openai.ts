// eslint-disable-next-line
import { createTemperatureWorkflowToolOpenai } from './workflow';

async function main() {
  // mcpTool validates Zod v4 is installed before loading @openai/agents, so call it first
  // to get a clear error message if the wrong Zod version is present.
  const temperatureTool = createTemperatureWorkflowToolOpenai();

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

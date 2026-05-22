import { tuningEnginesWorkflow } from './workflow';

async function main() {
  const result = await tuningEnginesWorkflow.run({
    prompt: 'Summarize why durable workflow retries are useful for AI tasks.',
  });

  console.log(result['governed-model-call']);
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => {
      process.exit(0);
    });
}

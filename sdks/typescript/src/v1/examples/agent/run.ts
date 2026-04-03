import { evaluatorOptimizerAgent } from './agents/effective-agent-patterns/4.evaluator-optimizer/evaluator-optimizer.agent';

async function main() {
  const result = await evaluatorOptimizerAgent.run({
    topic: 'a post about parallelization in python',
    targetAudience: 'senior developers',
  });
  console.log(JSON.stringify(result, null, 2));
}

main()
  .catch(console.error)
  .finally(() => {
    process.exit(0);
  });

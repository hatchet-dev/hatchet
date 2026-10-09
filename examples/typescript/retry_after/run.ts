import { retryAfterUpstreamDelay } from './workflow';

async function main() {
  const res = await retryAfterUpstreamDelay.run({ failingAttempts: 2 });

  console.log(res);
}

if (require.main === module) {
  main();
}

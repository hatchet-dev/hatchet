import { hatchet } from '../hatchet-client';
import { simple } from './workflow';

async function main() {
  const res = await hatchet.run<{ Message: string }, { step2: string }>(simple, {
    Message: 'hello',
  });

  // eslint-disable-next-line no-console
  console.log(res.step2);
}

if (require.main === module) {
  main();
}

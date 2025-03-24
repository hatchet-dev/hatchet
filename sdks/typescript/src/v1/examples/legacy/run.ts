import { hatchet } from '../client';
import { simple } from './workflow';

async function main() {
  const res = await hatchet.run(simple, {
    Message: 'hello',
  });

  // eslint-disable-next-line no-console
  console.log(res.step2);
}

if (require.main === module) {
  main();
}

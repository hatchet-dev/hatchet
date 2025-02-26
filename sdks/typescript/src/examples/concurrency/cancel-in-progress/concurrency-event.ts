import Hatchet from '../../../sdk';

const hatchet = Hatchet.init();

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

async function main() {
  hatchet.event.push('concurrency:create', {
    data: 'event 1',
    userId: 'user1',
  });

  // step 1 will wait 5000 ms,
  // so sending a second event
  // before that will cancel
  // the first run and run the second event
  await sleep(1000);

  hatchet.event.push('concurrency:create', {
    data: 'event 2',
    userId: 'user1',
  });
}

main();

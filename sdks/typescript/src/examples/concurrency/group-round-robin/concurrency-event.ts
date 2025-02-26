import Hatchet from '../../../sdk';

const hatchet = Hatchet.init();

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

async function main() {
  // eslint-disable-next-line no-plusplus
  for (let i = 0; i < 20; i++) {
    let group = 0;

    if (i > 10) {
      group = 1;
    }

    hatchet.event.push('concurrency:create', {
      data: `event ${i}`,
      group,
    });
  }
}

main();

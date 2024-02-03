import Hatchet from '../../src/sdk';

const hatchet = Hatchet.init();

hatchet.event.push('concurrency:create', {
  test: 'test',
});

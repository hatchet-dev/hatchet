import Hatchet from '../src/sdk';

const hatchet = Hatchet.init();

hatchet.event.push('user:create', {
  test: 'test',
});

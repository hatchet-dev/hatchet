import Hatchet from '@hatchet/sdk';

const hatchet = Hatchet.init();

hatchet.event.push('user:create', {
  test: 'test',
});

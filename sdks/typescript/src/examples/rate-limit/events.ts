import Hatchet from '../../sdk';

const hatchet = Hatchet.init();

hatchet.events.push('rate-limit:create', {
  test: '1',
});
hatchet.events.push('rate-limit:create', {
  test: '2',
});
hatchet.events.push('rate-limit:create', {
  test: '3',
});

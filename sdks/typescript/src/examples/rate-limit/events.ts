import Hatchet from '../../sdk';

const hatchet = Hatchet.init();

hatchet.event.push('rate-limit:create', {
  test: '1',
});
hatchet.event.push('rate-limit:create', {
  test: '2',
});
hatchet.event.push('rate-limit:create', {
  test: '3',
});

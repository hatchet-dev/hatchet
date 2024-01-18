import { Client } from '@clients/client';

const hatchet = new Client();

hatchet.event.push('user:create', {
  test: 'test',
});

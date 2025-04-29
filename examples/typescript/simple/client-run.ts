// > Client Run Methods
import { hatchet } from '../hatchet-client';

hatchet.run('simple', { Message: 'Hello, World!' });

hatchet.runNoWait('simple', { Message: 'Hello, World!' }, {});

hatchet.schedules.create('simple', {
  triggerAt: new Date(Date.now() + 1000 * 60 * 60 * 24),
  input: { Message: 'Hello, World!' },
});

hatchet.crons.create('simple', {
  name: 'my-cron',
  expression: '0 0 * * *',
  input: { Message: 'Hello, World!' },
});


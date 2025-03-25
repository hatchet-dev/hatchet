/* eslint-disable no-console */
import { hatchet } from '../hatchet-client';

// ❓ Simple Step Retries
export const retries = hatchet.task({
  name: 'retries',
  retries: 3,
  fn: async () => {
    throw new Error('intentional failure');
  },
});
// !!

// > Slot cost
import { hatchet } from '../hatchet-client';

export const omega = hatchet.task({
  name: 'omega',
  slotCost: 5,
  fn: async () => {
    console.log('heavy work');
  },
});

export const weenie = hatchet.task({
  name: 'weenie',
  slotCost: 1,
  fn: async () => {
    console.log('light work');
  },
});

// !!

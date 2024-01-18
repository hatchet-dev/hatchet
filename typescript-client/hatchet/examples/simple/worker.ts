import { Workflow } from '../../workflow';

const w: Workflow = {
  id: 'test',
  description: 'test',
  on: {
    cron: 'test',
  },
  steps: [
    {
      name: 'test',
      run: (input, ctx) => {
        return { test: 'test' };
      },
    },
  ],
};

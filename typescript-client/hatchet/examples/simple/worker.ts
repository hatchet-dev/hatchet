import Hatchet from '../..';
import { Workflow } from '../../workflow';

const hatchet = Hatchet.init();

const workflow: Workflow = {
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

// const worker = hatchet.worker(workflow);
// worker.start();

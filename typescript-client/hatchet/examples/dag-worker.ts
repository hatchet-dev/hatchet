import Hatchet from '..';
import { Workflow } from '../workflow';

const hatchet = Hatchet.init();

const workflow: Workflow = {
  id: 'dag-example',
  description: 'test',
  on: {
    event: 'user:create',
  },
  steps: [
    {
      name: 'step1',
      run: (input, ctx) => {
        console.log('executed step1!');
        return { step1: 'step1' };
      },
    },
    {
      name: 'step2',
      run: (input, ctx) => {
        console.log('executed step2!');
        return { step2: 'step2' };
      },
    },
    {
      name: 'step3',
      parents: ['step1', 'step2'],
      run: (input, ctx) => {
        console.log('executed step3!');
        return { step3: 'step3' };
      },
    },
    {
      name: 'step4',
      parents: ['step1', 'step3'],
      run: (input, ctx) => {
        console.log('executed step4!');
        return { step4: 'step4' };
      },
    },
  ],
};

const worker = hatchet.worker(workflow);
worker.start();

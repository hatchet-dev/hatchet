import sleep from '@hatchet/util/sleep';
import { hatchet } from '../client';

type DagInput = {};

type DagOutput = {
  'first-task': {
    Completed: boolean;
  };
  'second-task': {
    Completed: boolean;
  };
};

export const dagWithConditions = hatchet.workflow<DagInput, DagOutput>({
  name: 'simple',
});

const firstTask = dagWithConditions.task({
  name: 'first-task',
  fn: async () => {
    await sleep(2000);
    return {
      Completed: true,
    };
  },
});

dagWithConditions.task({
  name: 'second-task',
  parents: [firstTask],
  queueIf: { eventKey: 'user:update' },
  fn: async () => {
    return {
      Completed: true,
    };
  },
});

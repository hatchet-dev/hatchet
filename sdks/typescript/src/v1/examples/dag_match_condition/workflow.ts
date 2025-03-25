import sleep from '@hatchet/util/sleep';
import { Or } from '@hatchet/v1/conditions';
import { hatchet } from '../hatchet-client';

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
  waitFor: Or({ eventKey: 'user:update' }, { parent: firstTask }),
  fn: async () => {
    return {
      Completed: true,
    };
  },
});

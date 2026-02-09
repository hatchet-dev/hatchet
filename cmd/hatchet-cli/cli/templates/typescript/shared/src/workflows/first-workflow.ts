import { hatchet } from '../hatchet-client';

type Input = {
  message: string;
};

type Output = {
  "first-task": {
    message: string;
  };
  "second-task": {
    message: string;
  };
};

export const simple = hatchet.workflow<Input, Output>({
  name: 'first-workflow',
});

const step1 = simple.task({
  name: 'first-task',
  fn: (input) => {
    return {
      "message": "Hello, world!"
    };
  },
});

const step2 = simple.task({
  name: 'second-task',
  parents: [step1],
  fn: (input) => {
    return {
      "message": "Hello, moon!"
    };
  },
});

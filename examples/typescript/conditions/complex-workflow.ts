// > Create a workflow
import { Or, SleepCondition, UserEventCondition } from '@hatchet-dev/typescript-sdk/v1/conditions';
import { ParentCondition } from '@hatchet-dev/typescript-sdk/v1/conditions/parent-condition';
import { Context } from '@hatchet-dev/typescript-sdk/v1/client/worker/context';
import { hatchet } from '../hatchet-client';

export const taskConditionWorkflow = hatchet.workflow({
  name: 'TaskConditionWorkflow',
});

// > Add base task
const start = taskConditionWorkflow.task({
  name: 'start',
  fn: () => {
    return {
      randomNumber: Math.floor(Math.random() * 100) + 1,
    };
  },
});

// > Add wait for sleep
const waitForSleep = taskConditionWorkflow.task({
  name: 'waitForSleep',
  parents: [start],
  waitFor: [new SleepCondition('10s')],
  fn: () => {
    return {
      randomNumber: Math.floor(Math.random() * 100) + 1,
    };
  },
});

// > Add skip on event
const skipOnEvent = taskConditionWorkflow.task({
  name: 'skipOnEvent',
  parents: [start],
  waitFor: [new SleepCondition('10s')],
  skipIf: [new UserEventCondition('skip_on_event:skip', 'true')],
  fn: () => {
    return {
      randomNumber: Math.floor(Math.random() * 100) + 1,
    };
  },
});

// > Add branching
const leftBranch = taskConditionWorkflow.task({
  name: 'leftBranch',
  parents: [waitForSleep],
  skipIf: [new ParentCondition(waitForSleep, 'output.randomNumber > 50')],
  fn: () => {
    return {
      randomNumber: Math.floor(Math.random() * 100) + 1,
    };
  },
});

const rightBranch = taskConditionWorkflow.task({
  name: 'rightBranch',
  parents: [waitForSleep],
  skipIf: [new ParentCondition(waitForSleep, 'output.randomNumber <= 50')],
  fn: () => {
    return {
      randomNumber: Math.floor(Math.random() * 100) + 1,
    };
  },
});

// > Add wait for event
const waitForEvent = taskConditionWorkflow.task({
  name: 'waitForEvent',
  parents: [start],
  waitFor: [Or(new SleepCondition('1m'), new UserEventCondition('wait_for_event:start', 'true'))],
  fn: () => {
    return {
      randomNumber: Math.floor(Math.random() * 100) + 1,
    };
  },
});

// > Add sum
taskConditionWorkflow.task({
  name: 'sum',
  parents: [start, waitForSleep, waitForEvent, skipOnEvent, leftBranch, rightBranch],
  fn: async (_, ctx: Context<any, any>) => {
    const one = (await ctx.parentOutput(start)).randomNumber;
    const two = (await ctx.parentOutput(waitForEvent)).randomNumber;
    const three = (await ctx.parentOutput(waitForSleep)).randomNumber;
    const four = (await ctx.parentOutput(skipOnEvent))?.randomNumber || 0;
    const five = (await ctx.parentOutput(leftBranch))?.randomNumber || 0;
    const six = (await ctx.parentOutput(rightBranch))?.randomNumber || 0;

    return {
      sum: one + two + three + four + five + six,
    };
  },
});

import { Or, SleepCondition, UserEventCondition } from "@hatchet/v1/conditions";
import { hatchet } from "../hatchet-client";
import { ParentCondition } from "@hatchet/v1/conditions/parent-condition";
import { Context } from "@hatchet/step";

// Define types for our task outputs
interface StepOutput {
  randomNumber: number;
}

interface RandomSum {
  sum: number;
}

interface WorkflowOutput {
    start: StepOutput;
    waitForSleep: StepOutput;
    waitForEvent: StepOutput;
    skipOnEvent: StepOutput;
    leftBranch: StepOutput;
    rightBranch: StepOutput;
    sum: RandomSum;
}

// Create the workflow
export const taskConditionWorkflow = hatchet.workflow<never, WorkflowOutput>({
  name: 'TaskConditionWorkflow',
});

// Base task
const start = taskConditionWorkflow.task({
  name: 'start',
  fn: () => {
    return {
      randomNumber: Math.floor(Math.random() * 100) + 1
    } as StepOutput;
  }
});

// Wait for sleep task
const waitForSleep = taskConditionWorkflow.task({
  name: 'wait-for-sleep',
  parents: [start],
  waitFor: [new SleepCondition("10s")],
  fn: () => {
    return {
      randomNumber: Math.floor(Math.random() * 100) + 1
    } as StepOutput;
  }
});

// Skip on event task
const skipOnEvent = taskConditionWorkflow.task({
  name: 'skip-on-event',
  parents: [start],
  waitFor: [new SleepCondition("10s")],
  skipIf: [new UserEventCondition('skip_on_event:skip', "true")],
  fn: () => {
    return {
      randomNumber: Math.floor(Math.random() * 100) + 1
    } as StepOutput;
  }
});

// Left branch task
const leftBranch = taskConditionWorkflow.task({
  name: 'left-branch',
  parents: [waitForSleep],
  skipIf: [
    new ParentCondition(
      waitForSleep,
      'output.randomNumber > 50'
    )
  ],
  fn: () => {
    return {
      randomNumber: Math.floor(Math.random() * 100) + 1
    } as StepOutput;
  }
});

// Right branch task
const rightBranch = taskConditionWorkflow.task({
  name: 'right-branch',
  parents: [waitForSleep],
  skipIf: [
    new ParentCondition(
      waitForSleep,
      'output.randomNumber <= 50'
    )
  ],
  fn: () => {
    return {
      randomNumber: Math.floor(Math.random() * 100) + 1
    } as StepOutput;
  }
});

// Wait for event task
const waitForEvent = taskConditionWorkflow.task({
  name: 'wait-for-event',
  parents: [start],
  waitFor: [
    Or(
      new SleepCondition("1m"),
      new UserEventCondition('wait_for_event:start', "true")
    )
  ],
  fn: () => {
    return {
      randomNumber: Math.floor(Math.random() * 100) + 1
    } as StepOutput;
  }
});

// Sum task
taskConditionWorkflow.task({
  name: 'sum',
  parents: [
    start,
    waitForSleep,
    waitForEvent,
    skipOnEvent,
    leftBranch,
    rightBranch
  ],
  fn: async (_, ctx: Context<any, any>) => {
    const one = (await ctx.parentOutput(start)).randomNumber;
    const two = (await ctx.taskOutput<StepOutput>(waitForEvent)).randomNumber;
    const three = (await ctx.taskOutput<StepOutput>(waitForSleep)).randomNumber;
    const four = ctx.wasSkipped(skipOnEvent)
      ? 0
      : (await ctx.taskOutput<StepOutput>(skipOnEvent)).randomNumber;
    const five = ctx.wasSkipped(leftBranch)
      ? 0
      : (await ctx.taskOutput<StepOutput>(leftBranch)).randomNumber;
    const six = ctx.wasSkipped(rightBranch)
      ? 0
      : (await ctx.taskOutput<StepOutput>(rightBranch)).randomNumber;

    return {
      sum: one + two + three + four + five + six
    } as RandomSum;
  }
});

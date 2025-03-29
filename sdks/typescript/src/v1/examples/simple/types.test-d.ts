// eslint-disable-next-line import/no-extraneous-dependencies
import {
  CreateTaskWorkflowOpts,
  HatchetClient,
  TaskWorkflowDeclaration,
  UnknownInputType,
  WorkflowDeclaration,
} from '@hatchet/v1';
import { CreateWorkflowTaskOpts } from '@hatchet/v1/task';
// eslint-disable-next-line import/no-extraneous-dependencies
import { expectError, expectType } from 'jest-tsd';

// TODO
// return void

const hatchet = HatchetClient.init({ token: '' });

test('unknown input type', () => {
  const input: UnknownInputType = {};
  expectError(input.error);
});

test('task should propagate generics', () => {
  type In = { in: string };
  type Out = { out: string };

  const task = hatchet.task<In, Out>({
    name: '',
    fn: (input) => {
      console.log(input);
      return { out: 'string' };
    },
  });

  expectType<TaskWorkflowDeclaration<In, Out>>(task);
});

test('task infer from input and return type', () => {
  type In = { in: string };
  type Out = { out: string };

  const task = hatchet.task({
    name: '',
    fn: (input: In) => {
      console.log(input);
      return { out: 'string' } as Out;
    },
  });

  expectType<TaskWorkflowDeclaration<In, Out>>(task);
});

test('task should infer unknown, {} for an empty object', () => {
  const task = hatchet.task({
    name: '',
    fn: (input) => {
      return {};
    },
  });

  expectType<TaskWorkflowDeclaration<UnknownInputType, {}>>(task);
});

test('task should allow void', () => {
  const task = hatchet.task({
    name: '',
    fn: () => {},
  });

  expectType<TaskWorkflowDeclaration<unknown, void>>(task);
});

test('task should not allow non json object inputs', () => {
  type In = {
    NotSerializable: () => void;
  };

  const create: CreateTaskWorkflowOpts<In, void> = {
    name: '',
    fn: (input: In) => {
      console.log(input);
    },
  };

  expectError(create);
});

test('workflow should propagate generics', () => {
  type In = { in: string };
  type Out = { out: string };

  const workflow = hatchet.workflow<In, Out>({
    name: '',
  });

  expectType<WorkflowDeclaration<In, Out>>(workflow);

  const task = workflow.task({
    name: '',
    fn: () => {},
  });

  expectType<CreateWorkflowTaskOpts<In, void>>(task);
});

test('task infer from input and return type', () => {
  type In = { in: string };
  type Out = { out: string };

  const workflow = hatchet.workflow<In, Out>({
    name: '',
  });

  expectType<WorkflowDeclaration<In, Out>>(workflow);

  const task = workflow.task({
    name: '',
    fn: () => {},
  });

  expectType<CreateWorkflowTaskOpts<In, void>>(task);
});

// JsonObject restriction

test('task should infer unknown, {} for an empty object', () => {
  type In = { in: string };
  type Out = { out: string };

  const workflow = hatchet.workflow<In, Out>({
    name: '',
  });

  expectType<WorkflowDeclaration<In, Out>>(workflow);

  const task = workflow.task({
    name: '',
    fn: () => {},
  });

  expectType<CreateWorkflowTaskOpts<In, void>>(task);
});

test('task should allow void', () => {
  const task = hatchet.task({
    name: '',
    fn: () => {},
  });

  expectType<TaskWorkflowDeclaration<unknown, void>>(task);
});

// JsonObject restriction

// NOTE: this file will have lots of errors, but that's okay
// comment line    "**/*.test-d.ts" // NOTE: disable this if modifying types.test-d.ts
// in tsconfig.json to hide the errors (but fail on build)

// eslint-disable-next-line import/no-extraneous-dependencies
import { Context } from '@hatchet/step';
import {
  CreateTaskWorkflowOpts,
  HatchetClient,
  TaskOutputType,
  TaskWorkflowDeclaration,
  WorkflowDeclaration,
} from '@hatchet/v1';
import {
  CreateOnFailureTaskOpts,
  CreateOnSuccessTaskOpts,
  CreateWorkflowDurableTaskOpts,
  CreateWorkflowTaskOpts,
} from '@hatchet/v1/task';
// eslint-disable-next-line import/no-extraneous-dependencies
import { expectError, expectType } from 'jest-tsd';
import { UnknownInputType } from '@hatchet/v1/types';

const hatchet = HatchetClient.init({ token: '' });

test('unknown input type', () => {
  const input: UnknownInputType = {};
  expectError<UnknownInputType>(input.error);
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

test('task should infer output', () => {
  const task = hatchet.task({
    name: '',
    fn: () => {
      return { out: 'string' };
    },
  });

  expectType<TaskWorkflowDeclaration<UnknownInputType, { out: string }>>(task);
});

test('task should propagate generics', () => {
  type In = { in: string };
  type Out = { out: string };

  const task = hatchet.task<In, Out>({
    name: '',
    fn(
      input: { in: string },
      ctx: Context<{ in: string }, {}>
    ): { out: string } | Promise<{ out: string }> {
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

  expectType<TaskWorkflowDeclaration<UnknownInputType, void>>(task);
});

test('task should not allow non json object inputs', () => {
  type In = {
    NotSerializable: () => void;
  };

  expectError<CreateTaskWorkflowOpts<In, void>>({
    name: '',
    fn: (input: In) => {
      console.log(input);
    },
  });
});

test('workflow should propagate generics', () => {
  type In = { in: string };
  type Out = { out: { result: string } };

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
  type Out = { out: { result: string } };

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
  type Out = { out: { result: string } };

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

  expectType<TaskWorkflowDeclaration<UnknownInputType, void>>(task);
});

// JsonObject restriction

// Test durableTask type inference
test('durableTask should propagate generics', () => {
  type In = { in: string };
  type Out = { out: string };

  const task = hatchet.durableTask<In, Out>({
    name: '',
    fn: (input, ctx) => {
      console.log(input);
      return { out: 'string' };
    },
  });

  expectType<TaskWorkflowDeclaration<In, Out>>(task);
});

// Test onSuccess handler type inference
test('workflow onSuccess should inherit workflow input type', () => {
  type In = { in: string };
  type Out = { out: { result: string } };

  const workflow = hatchet.workflow<In, Out>({
    name: '',
  });

  const successHandler = workflow.onSuccess({
    fn: (input, ctx) => {
      expectType<In>(input);
      return { result: 'success' };
    },
  });

  expectType<CreateWorkflowTaskOpts<In, { result: string }>>(successHandler);
});

// Test onFailure handler type inference
test('workflow onFailure should inherit workflow input type', () => {
  type In = { in: string };
  type Out = { out: { result: string } };

  const workflow = hatchet.workflow<In, Out>({
    name: '',
  });

  const failureHandler = workflow.onFailure({
    fn: (input, ctx) => {
      expectType<In>(input);
      return { error: 'failed' };
    },
  });

  expectType<CreateWorkflowTaskOpts<In, { error: string }>>(failureHandler);
});

// Test TaskOutputType utility
test('TaskOutputType should extract the correct type', () => {
  type TaskResults = {
    task1: { data: string };
    task2: number[];
  };

  // Should extract the type from TaskResults when key exists
  type Task1Type = TaskOutputType<TaskResults, 'task1', never>;
  expectType<{ data: string }>({} as Task1Type);

  // Should fall back to the inferred type when key doesn't exist
  type Task3Type = TaskOutputType<TaskResults, 'task3', { result: boolean }>;
  expectType<{ result: boolean }>({} as Task3Type);
});

// Test array inputs to run methods
test('workflow run should handle array inputs', () => {
  type In = { id: number };
  type Out = { result: { result: string } };

  const workflow = hatchet.workflow<In, Out>({ name: '' });

  const runSingle = workflow.run({ id: 1 });
  expectType<Promise<Out>>(runSingle);

  const runMultiple = workflow.run([{ id: 1 }, { id: 2 }]);
  expectType<Promise<Out[]>>(runMultiple);

  const runAndWaitSingle = workflow.runAndWait({ id: 1 });
  expectType<Promise<Out>>(runAndWaitSingle);

  const runAndWaitMultiple = workflow.runAndWait([{ id: 1 }, { id: 2 }]);
  expectType<Promise<Out[]>>(runAndWaitMultiple);
});

// Test task result access in chained tasks
test('workflow task should be able to access results of parent tasks', () => {
  type In = { input: string };

  const workflow = hatchet.workflow<In>({
    name: '',
  });

  const task1 = workflow.task({
    name: 'task1',
    fn: () => ({ result1: 'data' }),
  });

  const task2 = workflow.task({
    name: 'task2',
    parents: [task1],
    fn: (input, ctx) => {
      // This is where we would access task1's result
      // Through ctx.results.task1
      return { result2: 'based on task1' };
    },
  });

  expectType<CreateWorkflowTaskOpts<In, { result1: string }>>(task1);
  expectType<CreateWorkflowTaskOpts<In, { result2: string }>>(task2);
});

// Test task result access in chained tasks
test('workflow durableTask should be able to access results of parent tasks', () => {
  type In = { input: string };

  const workflow = hatchet.workflow<In>({
    name: '',
  });

  const task1 = workflow.durableTask({
    name: 'task1',
    fn: () => ({ result1: 'data' }),
  });

  const task2 = workflow.durableTask({
    name: 'task2',
    parents: [task1],
    fn: (input, ctx) => {
      // This is where we would access task1's result
      // Through ctx.results.task1
      return { result2: 'based on task1' };
    },
  });

  expectType<CreateWorkflowTaskOpts<In, { result1: string }>>(task1);
  expectType<CreateWorkflowTaskOpts<In, { result2: string }>>(task2);
});

// Test complex nested object type preservation
test('complex nested object types should be preserved', () => {
  type ComplexInput = {
    user: {
      profile: {
        name: string;
        preferences: {
          theme: 'light' | 'dark';
          notifications: boolean;
        };
      };
      metadata: Record<string, string>;
    };
    items: Array<{
      id: number;
      tags: string[];
    }>;
  };

  const task = hatchet.task<ComplexInput, void>({
    name: 'complex',
    fn: (input) => {
      // Type checking should work for deeply nested properties
      const { theme } = input.user.profile.preferences;
      expectType<'light' | 'dark'>(theme);

      const firstItemId = input.items[0].id;
      expectType<number>(firstItemId);
    },
  });

  expectType<TaskWorkflowDeclaration<ComplexInput, void>>(task);
});

// Test serialization constraints directly on the type definitions
test('handler types should enforce serializable constraints', () => {
  type In = { in: string };

  // Test CreateOnSuccessTaskOpts constraints
  expectError<CreateOnSuccessTaskOpts<In, { handler: () => void }>>({
    fn: (input) => ({
      handler: () => console.log('not serializable'),
    }),
  });

  // Test CreateOnFailureTaskOpts constraints
  expectError<CreateOnFailureTaskOpts<In, { handler: () => void }>>({
    fn: (input) => ({
      handler: () => console.log('not serializable'),
    }),
  });

  // Non-serializable input type
  type NonSerializableInput = {
    func: () => void;
  };

  // Should error on non-serializable input
  expectError<CreateOnSuccessTaskOpts<NonSerializableInput, { result: string }>>({
    fn: (input) => {
      input.func(); // Should error because func is not serializable
      return { result: 'ok' };
    },
  });
});

// Test task type constraints directly
test('task type constraints should enforce serialization requirements', () => {
  // Non-serializable input
  type NonSerializableInput = {
    func: () => void;
  };

  // Should error on CreateTaskWorkflowOpts with non-serializable input
  expectError<CreateTaskWorkflowOpts<NonSerializableInput, { result: string }>>({
    name: 'task',
    fn: (input) => {
      return { result: 'done' };
    },
  });

  // Non-serializable output
  type NonSerializableOutput = {
    handler: () => void;
  };

  // Should error on CreateTaskWorkflowOpts with non-serializable output
  expectError<CreateTaskWorkflowOpts<{ id: string }, NonSerializableOutput>>({
    name: 'task',
    fn: (input) => {
      return {
        handler: () => console.log('not serializable'),
      };
    },
  });
});

// Test durableTask type constraints directly
test('durableTask type constraints should enforce serialization requirements', () => {
  // Non-serializable input
  type NonSerializableInput = {
    callback: () => void;
  };

  // Should error on CreateWorkflowDurableTaskOpts with non-serializable input
  expectError<CreateWorkflowDurableTaskOpts<NonSerializableInput, { result: string }>>({
    name: 'durableTask',
    fn: (input, ctx) => {
      return { result: 'done' };
    },
  });

  // Non-serializable output
  type NonSerializableOutput = {
    handler: () => void;
  };

  // Should error on CreateWorkflowDurableTaskOpts with non-serializable output
  expectError<CreateWorkflowDurableTaskOpts<{ id: string }, NonSerializableOutput>>({
    name: 'durableTask',
    fn: (input, ctx) => {
      return {
        handler: () => console.log('not serializable'),
      };
    },
  });
});

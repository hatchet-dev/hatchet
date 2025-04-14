/* eslint-disable no-underscore-dangle */
import { Context } from '@hatchet/step';
import { JsonObject } from '../../types';
import { BaseWorkflowDeclaration } from '../../declaration';
import { HatchetClient } from '../../client/client';

type SingleMiddleware<T> = (input: JsonObject) => T | Promise<T>;
type SingleSerialize<T> = (input: T) => JsonObject | Promise<JsonObject>;

export interface Serializable<T = any> {
  deserialize: SingleMiddleware<T>;
  serialize: SingleSerialize<T>;
}

export interface Middleware {
  input: Serializable;
  output: Serializable;
}

export interface MiddlewareChain {
  middlewares: Middleware[];
}

export type MiddlewareFn<T = any> = (input: any) => T | Promise<T>;

// Helper to compose middleware functions
const composeMiddleware = (fns: MiddlewareFn[]) => {
  return async (input: any) => {
    let result = input;
    for (const fn of fns) {
      result = await fn(result);
    }
    return result;
  };
};

// Get all deserialize functions in the correct order
const getDeserializeChain = (middleware: Middleware[]) => {
  return middleware.map((m) => m.input.deserialize);
};

// Get all serialize functions in the correct order
const getSerializeChain = (middleware: Middleware[]) => {
  return middleware.map((m) => m.input.serialize);
};

/**
 * Binds middleware to a workflow's tasks
 * @param wf - The workflow declaration to bind middleware to
 * @param client - The HatchetClient instance
 * @returns The workflow with middleware bound to its tasks
 */
export async function bindMiddleware(
  wf: BaseWorkflowDeclaration<any, any>,
  client: HatchetClient
): Promise<BaseWorkflowDeclaration<any, any>> {
  const tasks = wf.definition._tasks.map((task) => {
    if (!task.middleware) {
      // eslint-disable-next-line no-param-reassign
      task.middleware = [
        {
          input: {
            deserialize: (input: JsonObject) => input,
            serialize: (input: any) => input,
          },
          output: {
            deserialize: (input: JsonObject) => input,
            serialize: (input: any) => input,
          },
        },
      ];
    }

    const originalFn = task.fn;

    // Build input middleware chain
    let inputDeserializeChain: MiddlewareFn[] = [];
    let inputSerializeChain: MiddlewareFn[] = [];

    if (task.middleware) {
      inputDeserializeChain = getDeserializeChain(task.middleware);
      inputSerializeChain = getSerializeChain(task.middleware);
    }

    // Build output middleware chain
    let outputDeserializeChain: MiddlewareFn[] = [];
    let outputSerializeChain: MiddlewareFn[] = [];

    if (task.middleware) {
      outputDeserializeChain = getDeserializeChain(task.middleware);
      outputSerializeChain = getSerializeChain(task.middleware);
    }

    // Add client middleware if present
    if (client.middleware) {
      inputDeserializeChain = [...getDeserializeChain(client.middleware), ...inputDeserializeChain];
      inputSerializeChain = [...inputSerializeChain, ...getSerializeChain(client.middleware)];
      outputDeserializeChain = [
        ...getDeserializeChain(client.middleware),
        ...outputDeserializeChain,
      ];
      outputSerializeChain = [...outputSerializeChain, ...getSerializeChain(client.middleware)];
    }

    // Compose the middleware with the original function
    // eslint-disable-next-line no-param-reassign
    task.fn = async (input: JsonObject, ctx: Context<any, any>) => {
      console.log('fn input', input);
      // Apply input deserialize chain
      const deserializedInput = await composeMiddleware(inputDeserializeChain)(input);

      // Run the original function
      const result = await originalFn(deserializedInput, ctx);

      // Apply output serialize chain
      return composeMiddleware(outputSerializeChain)(result);
    };

    return task;
  });

  // eslint-disable-next-line no-param-reassign
  wf.definition._tasks = tasks;
  return wf;
}

export async function serializeInput(
  input: JsonObject,
  middleware?: Middleware[]
): Promise<JsonObject> {
  let serialized = input;
  if (middleware) {
    for (const m of middleware) {
      serialized = await m.input.serialize(serialized);
    }
  }
  return serialized;
}

export async function deserializeOutput(
  output: JsonObject,
  middleware?: Middleware[]
): Promise<JsonObject> {
  let deserialized = output;
  if (middleware) {
    for (const m of middleware) {
      deserialized = await m.output.deserialize(deserialized);
    }
  }
  return deserialized;
}

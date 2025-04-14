import { Context } from '@hatchet/step';

type SingleMiddleware<T> = (input: string, ctx: Context<any, any>) => T | Promise<T>;
type SingleSerialize<T> = (input: T, ctx: Context<any, any>) => string | Promise<string>;

type MiddlewareChain<T, R = any> =
  | [SingleMiddleware<T>]
  | [
      (input: string, ctx: Context<any, any>) => R | Promise<R>,
      ...((input: R, ctx: Context<any, any>) => R | Promise<R>)[],
      (input: R, ctx: Context<any, any>) => T | Promise<T>,
    ];

type SerializeChain<T> =
  | [SingleSerialize<T>]
  | [
      (input: T, ctx: Context<any, any>) => any | Promise<any>,
      ...((input: any, ctx: Context<any, any>) => any | Promise<any>)[],
      (input: any, ctx: Context<any, any>) => string | Promise<string>,
    ];

export interface Middleware<T = any> {
  deserialize: MiddlewareChain<T> | SingleMiddleware<T>;
  serialize: SerializeChain<T> | SingleSerialize<T>;
}

export type MiddlewareFn<T = any> = (input: any, ctx: Context<any, any>) => T | Promise<T>;

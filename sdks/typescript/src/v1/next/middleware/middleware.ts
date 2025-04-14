import { Context } from '@hatchet/step';

export interface Middleware {
  deserialize: MiddlewareFn | MiddlewareFn[];
  serialize: MiddlewareFn | MiddlewareFn[];
}

export type MiddlewareFn = (input: any, ctx: Context<any, any>) => Promise<any>;

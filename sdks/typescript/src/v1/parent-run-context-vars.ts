import { AsyncLocalStorage } from 'async_hooks';
import type { DurableContext } from './client/worker/context';

export interface ParentRunContext {
  parentId: string;
  /**
   * External ID of the parent task/step run.
   */
  parentTaskRunExternalId: string;
  desiredWorkerId: string;
  childIndex?: number;

  /**
   * (optional) AbortSignal inherited by nested `run()` calls.
   * Used to cancel local "wait for result" subscriptions when the parent task is cancelled.
   */
  signal?: AbortSignal;

  /**
   * Present when the current task is running in durable mode.
   * Used by child `run()` calls to route through `spawnChild` instead of a fresh trigger.
   */
  durableContext?: DurableContext<unknown, unknown>;
}

export class ParentRunContextManager {
  private storage: AsyncLocalStorage<ParentRunContext>;

  constructor() {
    this.storage = new AsyncLocalStorage<ParentRunContext>();
  }

  runWithContext<T>(opts: ParentRunContext, fn: () => T): T {
    return this.storage.run(
      {
        ...opts,
      },
      fn
    );
  }

  setContext(opts: ParentRunContext): void {
    this.storage.enterWith({
      ...opts,
    });
  }

  setParentRunIdAndIncrementChildIndex(opts: ParentRunContext): void {
    const parentRunContext = this.getContext();
    if (parentRunContext) {
      this.setContext({
        ...parentRunContext,
        parentId: opts.parentId,
        parentTaskRunExternalId: opts.parentTaskRunExternalId,
        childIndex: (parentRunContext.childIndex ?? 0) + 1,
      });
    }
  }

  incrementChildIndex(n: number): void {
    const parentRunContext = this.getContext();
    if (parentRunContext) {
      // Build a fresh object to avoid mutating the live store reference, which
      // would corrupt any values snapshotted before this call.
      this.setContext({
        ...parentRunContext,
        childIndex: (parentRunContext.childIndex ?? 0) + n,
      });
    }
  }

  getContext(): ParentRunContext | undefined {
    return this.storage.getStore();
  }
}

// Export a default instance for backward compatibility and convenience
export const parentRunContextManager = new ParentRunContextManager();

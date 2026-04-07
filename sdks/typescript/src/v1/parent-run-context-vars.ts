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
      parentRunContext.parentId = opts.parentId;
      parentRunContext.parentTaskRunExternalId = opts.parentTaskRunExternalId;
      parentRunContext.childIndex = (parentRunContext.childIndex ?? 0) + 1;
      this.setContext(parentRunContext);
    }
  }

  incrementChildIndex(n: number): void {
    const parentRunContext = this.getContext();
    if (parentRunContext) {
      // Mutate in place — do NOT use enterWith/setContext here.
      // storage.run() gives every async descendant the same object reference,
      // so direct mutation is visible across all await boundaries within the
      // same task execution.  enterWith would create a new object whose
      // updates are invisible to parent async contexts after an await.
      parentRunContext.childIndex = (parentRunContext.childIndex ?? 0) + n;
    }
  }

  getContext(): ParentRunContext | undefined {
    return this.storage.getStore();
  }
}

// Export a default instance for backward compatibility and convenience
export const parentRunContextManager = new ParentRunContextManager();

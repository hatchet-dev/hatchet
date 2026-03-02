import { AsyncLocalStorage } from 'async_hooks';

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
      parentRunContext.childIndex = (parentRunContext.childIndex ?? 0) + n;
      this.setContext(parentRunContext);
    }
  }

  getContext(): ParentRunContext | undefined {
    return this.storage.getStore();
  }
}

// Export a default instance for backward compatibility and convenience
export const parentRunContextManager = new ParentRunContextManager();

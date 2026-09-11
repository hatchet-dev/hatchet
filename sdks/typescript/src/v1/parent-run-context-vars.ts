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

/**
 * Where the manager keeps the context of the task currently executing. Node workers
 * install an `AsyncLocalStorage` backed store (see `parent-run-context-storage.ts`);
 * runtimes without async context tracking keep the default store, which never has a
 * context.
 */
export interface ParentRunContextStorage {
  run<T>(context: ParentRunContext, fn: () => T): T;
  getStore(): ParentRunContext | undefined;
}

const noContextStorage: ParentRunContextStorage = {
  run: (_context, fn) => fn(),
  getStore: () => undefined,
};

export class ParentRunContextManager {
  private storage: ParentRunContextStorage;

  constructor(storage: ParentRunContextStorage = noContextStorage) {
    this.storage = storage;
  }

  /**
   * Replaces the store used for subsequent `runWithContext` and `getContext` calls.
   */
  useStorage(storage: ParentRunContextStorage): void {
    this.storage = storage;
  }

  runWithContext<T>(opts: ParentRunContext, fn: () => T): T {
    return this.storage.run(
      {
        ...opts,
      },
      fn
    );
  }

  incrementChildIndex(n: number): void {
    const parentRunContext = this.getContext();
    if (parentRunContext) {
      // Mutate in place, do NOT use enterWith here.
      // storage.run() gives every async descendant the same object reference,
      // so direct mutation is visible across all await boundaries within the
      // same task execution.  enterWith would replace the object, and the new
      // object's updates would be invisible to parent async contexts after an await.
      parentRunContext.childIndex = (parentRunContext.childIndex ?? 0) + n;
    }
  }

  getContext(): ParentRunContext | undefined {
    return this.storage.getStore();
  }
}

// Export a default instance for backward compatibility and convenience
export const parentRunContextManager = new ParentRunContextManager();

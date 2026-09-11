import { AsyncLocalStorage } from 'async_hooks';
import {
  ParentRunContext,
  ParentRunContextStorage,
  parentRunContextManager,
} from './parent-run-context-vars';

/**
 * `AsyncLocalStorage` backed store for the parent run context, so a task's `run()`,
 * `spawnChild` and event pushes see the task they were called from across awaits.
 */
export function createAsyncLocalParentRunContextStorage(): ParentRunContextStorage {
  const storage = new AsyncLocalStorage<ParentRunContext>();
  return {
    run: (context, fn) => storage.run(context, fn),
    getStore: () => storage.getStore(),
  };
}

let installed = false;

/**
 * Installs the `AsyncLocalStorage` store on the shared manager. Idempotent; the worker
 * calls it at module load so it is in place before any task runs.
 */
export function installAsyncLocalParentRunContext(): void {
  if (installed) return;
  parentRunContextManager.useStorage(createAsyncLocalParentRunContextStorage());
  installed = true;
}

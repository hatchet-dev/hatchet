import { ParentRunContextManager, parentRunContextManager } from './parent-run-context-vars';
import { createAsyncLocalParentRunContextStorage } from './parent-run-context-storage';

const context = { parentId: 'p', parentTaskRunExternalId: 't', desiredWorkerId: 'w' };

describe('ParentRunContextManager', () => {
  it('has no context by default and runs the callback in place', () => {
    const manager = new ParentRunContextManager();

    expect(manager.getContext()).toBeUndefined();
    expect(manager.runWithContext(context, () => manager.getContext())).toBeUndefined();
    expect(() => manager.incrementChildIndex(1)).not.toThrow();
  });

  it('tracks the context across awaits once the AsyncLocalStorage store is installed', async () => {
    const manager = new ParentRunContextManager();
    manager.useStorage(createAsyncLocalParentRunContextStorage());

    const seen = await manager.runWithContext(context, async () => {
      await Promise.resolve();
      manager.incrementChildIndex(2);
      await new Promise((r) => {
        setTimeout(r, 0);
      });
      return manager.getContext();
    });

    expect(seen).toMatchObject({ ...context, childIndex: 2 });
    expect(manager.getContext()).toBeUndefined();
  });

  it('is installed on the shared manager by the worker module', async () => {
    await import('./client/worker/worker-internal');

    const seen = parentRunContextManager.runWithContext(context, () =>
      parentRunContextManager.getContext()
    );

    expect(seen).toMatchObject(context);
  });
});

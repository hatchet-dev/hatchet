import { AsyncLocalStorage } from 'async_hooks';

export interface ParentRunContext {
  parentId: string;
  parentRunId: string;
  desiredWorkerId: string;
  childIndex?: number;
}

export class ParentRunContextManager {
  private storage: AsyncLocalStorage<ParentRunContext>;

  constructor() {
    this.storage = new AsyncLocalStorage<ParentRunContext>();
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

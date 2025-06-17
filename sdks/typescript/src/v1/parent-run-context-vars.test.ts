import { parentRunContextManager } from './parent-run-context-vars';

const childFn = async () => parentRunContextManager.getContext();

const parentFn = async () => {
  const parentRunContext = parentRunContextManager.getContext();
  return { parent: parentRunContext, child: await childFn() };
};

const wrapperFn = (id: string) => {
  parentRunContextManager.setContext({
    parentRunId: id,
    parentId: id,
    desiredWorkerId: '',
  });
  return parentFn();
};

describe('parent run context vars', () => {
  it('should set and get parent run context', async () => {
    const [res1, res2] = await Promise.all([wrapperFn('123'), wrapperFn('456')]);
    expect(res1.parent).toEqual({ parentRunId: '123', parentId: '123', desiredWorkerId: '' });
    expect(res1.child).toEqual({ parentRunId: '123', parentId: '123', desiredWorkerId: '' });
    expect(res2.parent).toEqual({ parentRunId: '456', parentId: '456', desiredWorkerId: '' });
    expect(res2.child).toEqual({ parentRunId: '456', parentId: '456', desiredWorkerId: '' });
  });

  it('should be undefined if not set', async () => {
    const { parent, child } = await parentFn();
    expect(parent).toBeUndefined();
    expect(child).toBeUndefined();
  });
});

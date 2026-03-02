import { Streamable } from './pooled-child-listener-client';

describe('RunGrpcPooledListener Streamable', () => {
  it('rejects with AbortError and runs cleanup when aborted', async () => {
    const onCleanup = jest.fn();
    // eslint-disable-next-line func-names, no-empty-function
    const streamable = new Streamable((async function* () {})(), 'run-1', onCleanup);

    const ac = new AbortController();
    const p = streamable.get({ signal: ac.signal });
    ac.abort();

    await expect(p).rejects.toMatchObject({ name: 'AbortError' });
    expect(onCleanup).toHaveBeenCalled();
  });
});

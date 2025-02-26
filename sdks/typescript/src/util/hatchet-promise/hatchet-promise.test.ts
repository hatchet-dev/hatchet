import HatchetPromise from './hatchet-promise';

describe('HatchetPromise', () => {
  it('should resolve the original promise if not canceled', async () => {
    const hatchetPromise = new HatchetPromise(
      new Promise((resolve) => {
        setTimeout(() => resolve('RESOLVED'), 500);
      })
    );
    const result = await hatchetPromise.promise;
    expect(result).toEqual('RESOLVED');
  });
  it('should resolve the cancel promise if canceled', async () => {
    const hatchetPromise = new HatchetPromise(
      new Promise((resolve) => {
        setTimeout(() => resolve('RESOLVED'), 500);
      })
    );

    const result = hatchetPromise.promise;
    setTimeout(() => {
      hatchetPromise.cancel();
    }, 100);

    try {
      await result;
      expect(true).toEqual(false); // this should not be reached
    } catch (e) {
      expect(e).toEqual(undefined);
    }
  });
});

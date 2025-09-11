import { batch } from './batch';

describe('batch', () => {
  it('should handle empty payloads', async () => {
    const items: any[] = [];
    const result = batch(items, 2, 10);
    expect(result).toEqual([]);
  });

  it('should batch by number of elements', async () => {
    const items = [{ a: 1 }, { a: 2 }, { a: 3 }];

    const result = batch(items, 2, 100);
    expect(result).toEqual([
      { payloads: [{ a: 1 }, { a: 2 }], originalIndices: [0, 1], batchIndex: 0 },
      { payloads: [{ a: 3 }], originalIndices: [2], batchIndex: 1 },
    ]);
  });

  it('should batch by size', async () => {
    const largePayload = new Array(1).fill('a').join('');

    const items = [
      { a: largePayload },
      { a: largePayload },
      { a: largePayload },
      { a: largePayload },
      { a: largePayload },
      { a: 1 },
      { a: 2 },
    ];

    const result = batch(items, 10, 20);

    expect(result).toEqual([
      {
        payloads: [{ a: largePayload }, { a: largePayload }],
        originalIndices: [0, 1],
        batchIndex: 0,
      },
      {
        payloads: [{ a: largePayload }, { a: largePayload }],
        originalIndices: [2, 3],
        batchIndex: 1,
      },
      { payloads: [{ a: largePayload }, { a: 1 }], originalIndices: [4, 5], batchIndex: 2 },
      { payloads: [{ a: 2 }], originalIndices: [6], batchIndex: 3 },
    ]);
  });
});

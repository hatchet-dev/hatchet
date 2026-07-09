import { mapSlotRequestsPb } from './worker-internal';

describe('mapSlotRequestsPb', () => {
  it('maps a slotCost to a request against the default pool', () => {
    expect(mapSlotRequestsPb({ slotCost: 5 }, false)).toEqual({ default: 5 });
  });

  it('defaults to one default slot when slotCost is omitted', () => {
    expect(mapSlotRequestsPb({}, false)).toEqual({ default: 1 });
  });

  it('accepts a slotCost of 1', () => {
    expect(mapSlotRequestsPb({ slotCost: 1 }, false)).toEqual({ default: 1 });
  });

  it('rejects a slotCost of 0 or a negative slotCost', () => {
    expect(() => mapSlotRequestsPb({ slotCost: 0 }, false)).toThrow(/positive integer/);
    expect(() => mapSlotRequestsPb({ slotCost: -3 }, false)).toThrow(/positive integer/);
  });

  it('rejects a non-integer slotCost', () => {
    expect(() => mapSlotRequestsPb({ slotCost: 2.5 }, false)).toThrow(/positive integer/);
  });

  it('keeps durable tasks on the durable pool and does not apply slotCost', () => {
    expect(mapSlotRequestsPb({}, true)).toEqual({ durable: 1 });
    expect(mapSlotRequestsPb({ slotCost: 5 }, true)).toEqual({ durable: 1 });
  });

  it('honors an explicit internal slotRequests map when present', () => {
    expect(mapSlotRequestsPb({ slotRequests: { default: 3 } }, false)).toEqual({ default: 3 });
  });
});

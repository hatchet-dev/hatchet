import { serializePayload } from './serialize-payload';

// A \uD800-\uDFFF escape that is not itself an escaped backslash followed by "ud...".
const SURROGATE_ESCAPE = /(?<!\\)(?:\\\\)*\\u[dD][89a-fA-F][0-9a-fA-F]{2}/;

describe('serializePayload', () => {
  it.each([
    {
      name: 'no surrogates',
      input: { s: 'hello', n: 1, b: true, z: null, list: [1, 'two'] },
      expected: { s: 'hello', n: 1, b: true, z: null, list: [1, 'two'] },
    },
    {
      name: 'whole emoji',
      input: { s: 'hi \ud83d\ude00' },
      expected: { s: 'hi \ud83d\ude00' },
    },
    {
      name: 'lone high surrogate',
      input: { s: 'cut \ud83d' },
      expected: { s: 'cut \ufffd' },
    },
    {
      name: 'lone low surrogate',
      input: { s: '\ude00 tail' },
      expected: { s: '\ufffd tail' },
    },
    {
      name: 'lone surrogate after a whole pair',
      input: '\ud83d\ude00\ud83d',
      expected: '\ud83d\ude00\ufffd',
    },
    {
      name: 'top-level string',
      input: '\udbff',
      expected: '\ufffd',
    },
    {
      name: 'lone surrogates nested in arrays and objects',
      input: { list: ['ok', ['\ud83d']], obj: { deep: { s: 'x\udc00y' } } },
      expected: { list: ['ok', ['\ufffd']], obj: { deep: { s: 'x\ufffdy' } } },
    },
    {
      name: 'literal backslash followed by ud83d',
      input: { s: '\\ud83d', lone: '\ud83d' },
      expected: { s: '\\ud83d', lone: '\ufffd' },
    },
    {
      name: 'value with toJSON',
      input: { wrapped: { toJSON: () => ({ s: 'a\ud83d' }) } },
      expected: { wrapped: { s: 'a\ufffd' } },
    },
    {
      name: 'members JSON.stringify drops or nulls',
      input: { u: undefined, f: () => 1, list: [undefined, '\udfff'] },
      expected: { list: [null, '\ufffd'] },
    },
  ])('$name', ({ input, expected }) => {
    const json = serializePayload(input);

    expect(JSON.parse(json)).toEqual(expected);
    expect(json).not.toMatch(SURROGATE_ESCAPE);
  });

  it.each([
    { name: 'undefined', input: undefined },
    { name: 'function', input: () => 1 },
    { name: 'symbol', input: Symbol('s') },
  ])('returns what JSON.stringify returns for $name', ({ input }) => {
    expect(serializePayload(input)).toBe(JSON.stringify(input));
  });

  it('throws as JSON.stringify does for a BigInt', () => {
    expect(() => serializePayload({ n: BigInt(1) })).toThrow(TypeError);
  });
});

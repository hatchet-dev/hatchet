import { createCopy, parseCopy, ref } from './copy';
import { COPY_KEYS } from './copy-keys';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { describe, it } from 'node:test';

const source = readFileSync(new URL('../copy.mdx', import.meta.url), 'utf8');

describe('copy.mdx', () => {
  it('has exactly the keys in copy-keys.ts', () => {
    const keys = [...parseCopy(source).keys()];

    assert.deepEqual(
      keys.filter((k) => !(COPY_KEYS as readonly string[]).includes(k)),
      [],
      'keys in copy.mdx that copy-keys.ts does not list',
    );
    assert.deepEqual(
      COPY_KEYS.filter((k) => !keys.includes(k)),
      [],
      'keys in copy-keys.ts that copy.mdx is missing',
    );
  });

  it('has text for every key', () => {
    for (const [key, entry] of parseCopy(source)) {
      assert.ok(entry.length > 0, `${key} is empty`);
    }
  });
});

describe('parseCopy', () => {
  const entries = parseCopy(`{/* a comment
## not.a.key
*/}

# Section

## a.text
First line
second line.

## a.list
- One
- Two, which
  continues.
`);

  it('joins text lines and drops comments and sections', () => {
    assert.equal(entries.get('a.text'), 'First line second line.');
    assert.equal(entries.has('not.a.key'), false);
    assert.equal(entries.has('Section'), false);
  });

  it('reads bulleted lists with continuation lines', () => {
    assert.deepEqual(entries.get('a.list'), ['One', 'Two, which continues.']);
  });
});

describe('createCopy', () => {
  const { resolveText, resolveList } = createCopy(
    new Map<string, string | string[]>([
      ['create.titleForPlatform', 'Add a [[platform]] endpoint'],
      ['platform.VERCEL.label', 'Vercel'],
      ['diagnosis.notPolled.steps.oss', ['Wait [[n]]s.', 'Check logs.']],
    ]),
  );

  it('fills placeholders, resolving nested copy', () => {
    assert.equal(
      resolveText(
        ref('create.titleForPlatform', {
          platform: ref('platform.VERCEL.label'),
        }),
      ),
      'Add a Vercel endpoint',
    );
  });

  it('keeps unknown placeholders and falls back to the key', () => {
    assert.equal(
      resolveText(ref('create.titleForPlatform')),
      'Add a [[platform]] endpoint',
    );
    assert.equal(resolveText(ref('list.add')), 'list.add');
  });

  it('resolves lists item by item', () => {
    assert.deepEqual(
      resolveList(ref('diagnosis.notPolled.steps.oss', { n: 5 })),
      ['Wait 5s.', 'Check logs.'],
    );
  });
});

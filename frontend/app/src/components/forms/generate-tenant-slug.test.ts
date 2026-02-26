import { generateTenantSlug } from './generate-tenant-slug';
import * as assert from 'node:assert';
import { test } from 'node:test';

test('generateTenantSlug', () => {
  const input1 = 'My Test Tenant';
  const input2 = 'Another Tenant';

  const slug1 = generateTenantSlug(input1);
  const slug2a = generateTenantSlug(input2);
  const slug2b = generateTenantSlug(input2);

  const slugPattern = /^[a-z0-9-]{6,}$/;

  assert.notStrictEqual(
    slug1,
    slug2a,
    'different inputs should produce different slugs',
  );
  assert.match(
    slug1,
    slugPattern,
    'slug from first input should match expected pattern',
  );
  assert.match(
    slug2a,
    slugPattern,
    'slug from second input should match expected pattern',
  );
  assert.match(
    slug2b,
    slugPattern,
    'slug from second input should match expected pattern',
  );

  assert.notStrictEqual(
    slug2a,
    slug2b,
    "same input should produce different slugs due to random suffix.  If this test ever fails, you still shouldn't buy a lottery ticket, but you should tell Josh, he'll think it's funny",
  );
});

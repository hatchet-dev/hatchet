import { generateTenantSlug } from './generate-tenant-slug';
import * as assert from 'node:assert';
import { test } from 'node:test';

test('generateTenantSlug', () => {
  const input_1 = 'My Test Tenant';
  const input_2 = 'Another Tenant';

  const slug_1 = generateTenantSlug(input_1);
  const slug_2a = generateTenantSlug(input_2);
  const slug_2b = generateTenantSlug(input_2);

  const slug_pattern = /^[a-z0-9-]{6,}$/;

  assert.notStrictEqual(
    slug_1,
    slug_2a,
    'different inputs should produce different slugs',
  );
  assert.match(
    slug_1,
    slug_pattern,
    'slug from first input should match expected pattern',
  );
  assert.match(
    slug_2a,
    slug_pattern,
    'slug from second input should match expected pattern',
  );
  assert.match(
    slug_2b,
    slug_pattern,
    'slug from second input should match expected pattern',
  );

  assert.notStrictEqual(
    slug_2a,
    slug_2b,
    "same input should produce different slugs due to random suffix.  If this test ever fails, you still shouldn't buy a lottery ticket, but you should tell Josh, he'll think it's funny",
  );
});

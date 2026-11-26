import {
  expect,
  login,
  pathnameOf,
  tenantIdFromUrl,
  test,
} from '../support/fixtures';
import type { Page } from '@playwright/test';

const nonExistentId = '00000000-0000-0000-0000-000000000000';

async function currentTenant(page: Page): Promise<string> {
  await expect(page).toHaveURL(/\/tenants\/[^/]+/, { timeout: 30_000 });
  return tenantIdFromUrl(page);
}

function expectText(page: Page, text: string) {
  return expect(page.getByText(text).first()).toBeVisible();
}

test.describe('Error pages', () => {
  test.describe('authenticated routes', () => {
    test.beforeEach(async ({ page }) => {
      await login(page, 'owner');
      await page.goto('/');
    });

    test('shows tenant forbidden when tenant param is not in memberships', async ({
      page,
    }) => {
      await page.goto('/tenants/missing-tenant/runs');
      await expectText(page, 'Access denied');
      await expectText(page, 'Requested Tenant');
    });

    test('shows workflow not found for invalid workflow ID', async ({
      page,
    }) => {
      const tenant = await currentTenant(page);
      await page.goto(`/tenants/${tenant}/workflows/${nonExistentId}`);
      await expectText(page, 'Workflow not found');
    });

    test('shows run not found for invalid run ID', async ({ page }) => {
      const tenant = await currentTenant(page);
      await page.goto(`/tenants/${tenant}/runs/${nonExistentId}`);
      await expectText(page, 'Run not found');
    });

    test('shows run not found for malformed run ID (e.g. UUID with extra char)', async ({
      page,
    }) => {
      const malformedRunId = '398948ac-18a9-44bc-8c57-b41d399e48cx';
      const tenant = await currentTenant(page);
      await page.goto(`/tenants/${tenant}/runs/${malformedRunId}`);
      await expectText(page, 'Run not found');
    });

    test('shows worker not found for invalid worker ID', async ({ page }) => {
      const tenant = await currentTenant(page);
      await page.goto(`/tenants/${tenant}/workers/${nonExistentId}`);
      await expectText(page, 'Worker not found');
    });

    test('shows custom 404 for unknown unnested routes', async ({ page }) => {
      await page.goto('/x');
      await expectText(page, 'Page not found');
    });

    test('shows custom 404 for unknown nested routes', async ({ page }) => {
      // Wait for the root redirect to settle on a tenant sub-route so the
      // appended "x" lands on an unknown nested path, not the tenant segment.
      await expect(page).toHaveURL(/\/tenants\/[^/]+\/[^/]+/, {
        timeout: 30_000,
      });
      await page.goto(`${pathnameOf(page)}x`);
      await expectText(page, 'Page not found');
    });
  });
});

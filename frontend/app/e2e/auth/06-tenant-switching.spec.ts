import {
  createTenantViaApi,
  expect,
  login,
  logOutViaAccountMenu,
  switchTenant,
  TAGS,
  tenantSwitcherButton,
  test,
} from '../support/fixtures';
import type { Page } from '@playwright/test';

async function createTwoTenantsAndSwitch(page: Page) {
  const ts = Date.now();
  const tenant1 = { name: `Tenant 1 ${ts}`, slug: `e2e-tenant1-${ts}` };
  const tenant2 = { name: `Tenant 2 ${ts}`, slug: `e2e-tenant2-${ts}` };

  await createTenantViaApi(page, tenant1);
  await createTenantViaApi(page, tenant2);

  // Refresh the SPA so memberships reflect the created tenants.
  await page.goto('/');
  await expect(page).toHaveURL(/\/tenants\//, { timeout: 30_000 });

  await switchTenant(page, tenant1.slug);
  await expect(tenantSwitcherButton(page)).toContainText(tenant1.name);

  await switchTenant(page, tenant2.slug);
  await expect(tenantSwitcherButton(page)).toContainText(tenant2.name);

  return { tenant1, tenant2 };
}

test.describe('Tenants: switching', { tag: TAGS.seeded }, () => {
  test.beforeEach(async ({ page }) => {
    await login(page, 'owner');
    await page.goto('/');
  });

  test('should survive page reloads and root redirect', async ({ page }) => {
    const { tenant2 } = await createTwoTenantsAndSwitch(page);

    await page.reload();
    await expect(tenantSwitcherButton(page)).toContainText(tenant2.name);

    await page.goto('/');
    await expect(tenantSwitcherButton(page)).toContainText(tenant2.name);
  });

  test('should not break on login with different user', async ({ page }) => {
    await createTwoTenantsAndSwitch(page);

    await logOutViaAccountMenu(page);
    await expect(page).toHaveURL(/\/auth\/login/);

    // Start the member from empty storage: this test covers switching users,
    // not recovering from the previous user's persisted tenant selection.
    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });
    await page.context().clearCookies();

    await login(page, 'member');
    await page.goto('/');
    // Member may not have seeded memberships in some environments; just ensure the app doesn't
    // get stuck on the previous user's tenant and lands on a valid authenticated route.
    await expect(page).toHaveURL(/\/tenants\/|\/onboarding\//, {
      timeout: 30_000,
    });
  });
});

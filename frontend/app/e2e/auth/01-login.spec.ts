import {
  completeCreateTenantOnboarding,
  expect,
  pathnameOf,
  signInWithForm,
  TAGS,
  test,
} from '../support/fixtures';

test.describe('auth: login', { tag: TAGS.seeded }, () => {
  test('should login a user with username and password', async ({ page }) => {
    await page.goto('/');
    await signInWithForm(page, 'owner');

    await expect(page).toHaveURL(/\/tenants\/|\/onboarding\/create-tenant/, {
      timeout: 30_000,
    });

    if (pathnameOf(page).includes('/onboarding/create-tenant')) {
      await completeCreateTenantOnboarding(
        page,
        `E2ELoginTenant${String(Date.now()).slice(-6)}`,
      );
    }

    await expect(page).toHaveURL(/\/tenants\//, { timeout: 30_000 });
    await expect(page.getByTestId('v1-sidebar')).toBeVisible({
      timeout: 30_000,
    });
  });
});

import {
  createTenantViaApi,
  declineRemainingInvites,
  expect,
  logOutViaAccountMenu,
  seededUser,
  signInWithForm,
  switchTenant,
  TAGS,
  test,
  waitForTenantShell,
} from '../support/fixtures';

test.describe('Tenant Invite: decline', { tag: TAGS.seeded }, () => {
  test('should close the invite modal after declining all invites', async ({
    page,
  }) => {
    const ts = Date.now();
    const tenantName = `DeclineTenant${ts}`;
    const tenantSlug = `decline-tenant-${ts}`;

    await page.goto('/auth/login');
    await signInWithForm(page, 'owner');
    await waitForTenantShell(page);

    const tenantId = await createTenantViaApi(page, {
      name: tenantName,
      slug: tenantSlug,
    });

    // Refresh so the tenant switcher sees the new tenant.
    await page.goto('/');
    await waitForTenantShell(page);

    await switchTenant(page, tenantSlug);
    await expect(page).toHaveURL(new RegExp(`/tenants/${tenantId}`), {
      timeout: 30_000,
    });

    const invite = await page.request.post(
      `/api/v1/tenants/${tenantId}/invites`,
      { data: { email: seededUser('member').email, role: 'MEMBER' } },
    );
    expect(invite.status()).toBe(201);

    await logOutViaAccountMenu(page);
    await expect(page).toHaveURL(/\/auth\/login/);

    await signInWithForm(page, 'member');
    await waitForTenantShell(page);

    // The invite modal auto-opens when the member has pending invites.
    await expect(page.locator('[role="dialog"]')).toBeVisible({
      timeout: 10_000,
    });
    await expect(page.locator('td', { hasText: tenantName })).toBeVisible();

    await declineRemainingInvites(page, { expectRejectStatus: 200 });

    await expect(page.locator('[role="dialog"]')).toHaveCount(0, {
      timeout: 10_000,
    });

    // The modal closes in place; no redirect.
    await expect(page).toHaveURL(/\/tenants\/.+/, { timeout: 10_000 });
  });
});

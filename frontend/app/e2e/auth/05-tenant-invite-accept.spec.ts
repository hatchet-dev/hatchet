import {
  createTenantViaApi,
  declineRemainingInvites,
  expect,
  logOutViaAccountMenu,
  signInWithForm,
  switchTenant,
  tenantSwitcherButton,
  test,
  waitForTenantShell,
} from '../support/fixtures';
import { seededUsers } from '../support/seeded-users.generated';

test.describe('Tenant Invite: accept', () => {
  test('should redirect to tenant page after accepting invite', async ({
    page,
  }) => {
    const ts = Date.now();
    const tenant2Name = `Tenant 2 ${ts}`;
    const tenant2Slug = `e2e-tenant2-${ts}`;

    await page.goto('/auth/login');
    await signInWithForm(page, 'owner');
    await waitForTenantShell(page);

    // Ensure Tenant 2 exists even if the DB isn't pre-seeded.
    const tenant2Id = await createTenantViaApi(page, {
      name: tenant2Name,
      slug: tenant2Slug,
    });

    // Refresh so the tenant switcher sees the updated memberships list.
    await page.goto('/');
    await waitForTenantShell(page);

    await switchTenant(page, tenant2Slug);
    await expect(page).toHaveURL(new RegExp(`/tenants/${tenant2Id}`), {
      timeout: 30_000,
    });

    const invite = await page.request.post(
      `/api/v1/tenants/${tenant2Id}/invites`,
      { data: { email: seededUsers.member.email, role: 'MEMBER' } },
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

    const accepted = page.waitForResponse(
      (r) =>
        r.request().method() === 'POST' &&
        r.url().includes('/api/v1/users/invites/accept'),
    );
    await page
      .locator('tr', { has: page.locator('td', { hasText: tenant2Name }) })
      .locator('button[aria-label="Accept"]')
      .click();
    expect((await accepted).status()).toBe(200);

    // The accepted row disappears once React flushes the state update; only
    // then is it safe to look for the remaining Decline buttons.
    await expect(page.locator('td', { hasText: tenant2Name })).toHaveCount(0);

    // Defensive: other specs may have left invites for the member.
    await declineRemainingInvites(page);

    await page.getByText(`Switch to ${tenant2Name}`).click({ timeout: 10_000 });

    await expect(page).toHaveURL(new RegExp(`/tenants/${tenant2Id}`));
    await expect(tenantSwitcherButton(page)).toContainText(tenant2Name);
  });
});

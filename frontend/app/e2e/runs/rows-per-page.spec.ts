import { expect, login, TAGS, tenantBasePath, test } from '../support/fixtures';
import type { Request } from '@playwright/test';

function isRunsListRequest(request: Request) {
  return (
    request.method() === 'GET' &&
    /^\/api\/v1\/stable\/tenants\/[^/]+\/workflow-runs$/.test(
      new URL(request.url()).pathname,
    )
  );
}

test.describe('runs table: rows per page', { tag: TAGS.seeded }, () => {
  test('changes rows per page and persists the change in the URL', async ({
    page,
  }) => {
    await login(page, 'owner');

    // Stub the workflow-count probe so the tenant looks onboarded and the
    // runs table (not the no-workflows placeholder) renders.
    await page.route(
      (url) =>
        /^\/api\/v1\/tenants\/[^/]+\/workflows$/.test(url.pathname) &&
        url.searchParams.get('limit') === '1',
      (route) =>
        route.fulfill({
          status: 200,
          json: {
            rows: [
              {
                metadata: {
                  id: 'a0000000-0000-0000-0000-000000000001',
                  createdAt: '2026-01-01T00:00:00.000Z',
                  updatedAt: '2026-01-01T00:00:00.000Z',
                },
                name: 'test-workflow',
              },
            ],
            pagination: { num_pages: 1 },
          },
        }),
    );

    await page.goto('/');
    await expect(page).toHaveURL(/\/tenants\/.+/, { timeout: 30_000 });

    // The recent-runs probe only fires for tenants with no workflows, so the
    // first list request is the table's own fetch at the default page size.
    const firstList = page.waitForRequest(isRunsListRequest);
    await page.goto(`${tenantBasePath(page)}/runs`);
    expect((await firstList).url()).toContain('limit=50');

    const rowsPerPage = page.locator('#rows-per-page');
    await rowsPerPage.click();

    const resized = page.waitForRequest(isRunsListRequest);
    await page.locator('[role="option"]', { hasText: /^25$/ }).click();
    expect((await resized).url()).toContain('limit=25');

    await expect(rowsPerPage).toContainText('25');

    // Pagination state is stored in a JSON-encoded query param.
    await expect
      .poll(() => {
        const raw = new URL(page.url()).searchParams.get(
          'pagination-runs-table-workflow-runs-main',
        );
        return raw ? (JSON.parse(raw) as { i: number; s: number }).s : null;
      })
      .toBe(25);
  });
});

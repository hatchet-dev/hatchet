import { expect, login, tenantBasePath, test } from '../support/fixtures';

test.describe('side panel: full height', () => {
  test('scheduled run side panel fills the available vertical space', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await login(page, 'owner');

    const workflowId = 'a0000000-0000-0000-0000-000000000001';
    const scheduledRunId = 'b0000000-0000-0000-0000-000000000002';
    const now = '2026-01-01T00:00:00.000Z';

    // Stub a registered workflow so the no-workflows onboarding placeholder
    // does not replace the scheduled runs table.
    await page.route(
      (url) => /^\/api\/v1\/tenants\/[^/]+\/workflows$/.test(url.pathname),
      (route) =>
        route.fulfill({
          status: 200,
          json: {
            rows: [
              {
                metadata: { id: workflowId, createdAt: now, updatedAt: now },
                name: 'test-workflow',
              },
            ],
            pagination: { num_pages: 1 },
          },
        }),
    );

    // Stub a scheduled run so a row is available to open the side panel.
    await page.route(
      (url) =>
        /^\/api\/v1\/tenants\/[^/]+\/workflows\/scheduled$/.test(url.pathname),
      (route) =>
        route.fulfill({
          status: 200,
          json: {
            rows: [
              {
                metadata: {
                  id: scheduledRunId,
                  createdAt: now,
                  updatedAt: now,
                },
                tenantId: '00000000-0000-0000-0000-000000000000',
                workflowVersionId: workflowId,
                workflowId,
                workflowName: 'test-workflow',
                triggerAt: '2026-12-31T00:00:00.000Z',
                method: 'API',
              },
            ],
            pagination: { num_pages: 1 },
          },
        }),
    );

    await page.goto('/');
    await expect(page).toHaveURL(/\/tenants\/.+/, { timeout: 30_000 });
    await page.goto(`${tenantBasePath(page)}/scheduled`);

    // Clicking the run row opens the scheduled run details side panel.
    await page.getByText(scheduledRunId).first().click();

    const panel = page.getByTestId('side-panel');
    await expect(panel).toBeVisible();

    // The panel content should fill the available vertical space, extending
    // to the bottom of the side panel. Polled because the panel animates in.
    const content = page.getByTestId('side-panel-content');
    await expect
      .poll(
        async () => {
          const panelBox = await panel.boundingBox();
          const contentBox = await content.boundingBox();
          if (!panelBox || !contentBox) {
            return Number.POSITIVE_INFINITY;
          }
          return Math.abs(
            contentBox.y + contentBox.height - (panelBox.y + panelBox.height),
          );
        },
        { message: 'content bottom offset from panel bottom' },
      )
      .toBeLessThanOrEqual(2);

    const contentBox = await content.boundingBox();
    expect(contentBox?.height ?? 0, 'content height').toBeGreaterThan(500);
  });
});

describe('runs table: rows per page', () => {
  it('changes rows per page and persists the change in the URL', () => {
    cy.login('owner');

    // Runs list request we expect to change when page size changes.
    cy.intercept('GET', '/api/v1/stable/tenants/*/workflow-runs*').as(
      'listRuns',
    );

    // Stub the workflow-count probe so the tenant looks onboarded and the
    // runs table (not the no-workflows placeholder) renders.
    cy.intercept(
      {
        method: 'GET',
        pathname: '/api/v1/tenants/*/workflows',
        query: { limit: '1' },
      },
      {
        statusCode: 200,
        body: {
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
      },
    );

    // Start from the authenticated root, then derive the active tenant base path.
    cy.visit('/');
    cy.location('pathname', { timeout: 30000 })
      .should('match', /\/tenants\/.+/)
      .then((pathname) => {
        const tenantBase = pathname.match(/\/tenants\/[^/]+/)?.[0];
        if (!tenantBase) {
          throw new Error(
            `Failed to derive tenant base path from: ${pathname}`,
          );
        }

        cy.visit(`${tenantBase}/runs`);
      });

    // The runs page first issues a limit=1 probe for recent runs (used to
    // decide whether to show the onboarding placeholder), then the table
    // loads with the default page size of 50.
    cy.wait('@listRuns').its('request.url').should('include', 'limit=1');
    cy.wait('@listRuns').its('request.url').should('include', 'limit=50');

    cy.get('#rows-per-page').should('be.visible').click();
    cy.contains('[role="option"]', '25').click();

    // Changing page size should refetch the list with a new limit.
    cy.wait('@listRuns').its('request.url').should('include', 'limit=25');

    // UI should reflect the new selection.
    cy.get('#rows-per-page').should('contain', '25');

    // Pagination state is stored in a JSON-encoded query param.
    cy.location('search').then((search) => {
      const params = new URLSearchParams(search);
      const raw = params.get('pagination-runs-table-workflow-runs-main');
      if (!raw) {
        throw new Error(
          'Missing pagination-runs-table-workflow-runs-main param',
        );
      }

      const parsed = JSON.parse(raw) as { i: number; s: number };
      assert.equal(parsed.s, 25);
    });
  });
});

describe('runs table: rows per page', () => {
  it('changes rows per page and persists the change in the URL', () => {
    cy.login('owner');

    // Runs list request we expect to change when page size changes.
    cy.intercept('GET', '/api/v1/stable/tenants/*/workflow-runs*').as(
      'listRuns',
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

    // Wait for the initial load (default page size is 50).
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

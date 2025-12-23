describe('side panel: full height', () => {
  it('docs side panel fills the available vertical space', () => {
    cy.viewport(1280, 800);
    cy.login('owner');

    // Force an empty state so the "Learn about scheduled runs" docs button is always present.
    cy.intercept('GET', '/api/v1/tenants/*/workflows/scheduled*', {
      statusCode: 200,
      body: { rows: [], pagination: { num_pages: 1 } },
    }).as('scheduledRuns');

    cy.intercept('GET', '/api/v1/tenants/*/workflows*', {
      statusCode: 200,
      body: { rows: [], pagination: { num_pages: 1 } },
    }).as('workflows');

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

        cy.visit(`${tenantBase}/scheduled`);
      });

    cy.wait(['@scheduledRuns', '@workflows']);

    cy.contains('button', 'Learn about scheduled runs')
      .should('be.visible')
      .click();

    cy.get('[data-cy="side-panel"]').should('be.visible');

    // And the docs iframe should fill the content box (excluding padding).
    cy.get('[data-cy="side-panel-content"] iframe').then(($iframe) => {
      const iframeH = $iframe[0].getBoundingClientRect().height;
      expect(iframeH, 'iframe height').to.be.closeTo(570, 10);
    });
  });
});

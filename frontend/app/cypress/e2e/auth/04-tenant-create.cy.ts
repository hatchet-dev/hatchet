describe('Tenants: create', () => {
  it('creates a new tenant', () => {
    const ts = Date.now();
    cy.login('owner');
    cy.visit('/');
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .click();
    cy.get('[data-cy="new-tenant"]').click();

    cy.get('[role="dialog"]', { timeout: 5000 }).should('exist');

    const tenantName = `CypressTenant${String(ts).slice(-6)}`;
    cy.get('[role="dialog"] #tenant-name')
      .filter(':visible')
      .first()
      .clear()
      .type(tenantName);
    cy.intercept('POST', '/api/v1/tenants').as('createTenant');
    cy.get('[role="dialog"] button[type="submit"]').click();
    cy.wait('@createTenant').its('response.statusCode').should('eq', 200);

    cy.location('pathname', { timeout: 30000 }).should(
      'match',
      /\/tenants\/.+\/overview/,
    );
    cy.contains('h1', 'Overview', { timeout: 30000 }).should('be.visible');

    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', tenantName);
  });
});

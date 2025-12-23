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

    // Authenticated layout redirects users without memberships to onboarding/create-tenant.
    cy.location('pathname', { timeout: 5000 }).should(
      'include',
      '/onboarding/create-tenant',
    );

    cy.contains('span', 'AI Agents').click();
    cy.contains('button', 'Continue').click();
    cy.contains('span', 'Hacker News').click();
    cy.contains('button', 'Continue').click();

    const tenantName = `CypressTenant${String(ts).slice(-6)}`;
    cy.get('input#name').clear().type(tenantName);
    cy.intercept('POST', '/api/v1/tenants').as('createTenant');
    cy.contains('button', 'Create Tenant').click();
    cy.wait('@createTenant').its('response.statusCode').should('eq', 200);

    cy.location('pathname', { timeout: 30000 }).should(
      'match',
      /\/tenants\/.+\/onboarding\/get-started/,
    );
    cy.contains('h1', 'Setup your', { timeout: 30000 }).should('be.visible');

    cy.get('button[aria-label="Select a tenant"]').should(
      'have.text',
      tenantName,
    );
  });
});

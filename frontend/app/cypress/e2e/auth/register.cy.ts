describe('Onboarding: create tenant', () => {
  it('registers a new user and creates a tenant', () => {
    const ts = Date.now();
    const userName = `Cypress User ${ts}`;
    const email = `cypress+${ts}@example.com`;
    // Password must include upper+lower+number.
    const password = `Cypress${ts}pw1`;
    // const tenantName = `CypressTenant${String(ts).slice(-6)}`;

    cy.intercept('POST', '/api/v1/users/register').as('register');
    cy.visit('/auth/register');

    cy.get('input#name').should('be.visible').clear().type(userName);
    cy.get('input#email').clear().type(email);
    cy.get('input#password').clear().type(password);
    cy.contains('button', 'Create Account').click();

    cy.wait('@register').its('response.statusCode').should('eq', 200);

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
  });
});

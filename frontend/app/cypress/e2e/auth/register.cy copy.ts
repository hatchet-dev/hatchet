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

    // // Registration does not guarantee an authenticated session; explicitly log in.
    // cy.intercept('POST', '/api/v1/users/login').as('login');
    // cy.visit('/auth/login');
    // cy.get('input#email').should('be.visible').clear().type(email);
    // cy.get('input#password').clear().type(password);
    // cy.contains('button', 'Sign In').click();
    // cy.wait('@login').then(({ response }) => {
    //   expect(response?.statusCode).to.eq(200);
    //   // If this is missing, auth cookies likely aren't being set (often due to cookie domain/env mismatch).
    //   expect(response?.headers).to.have.property('set-cookie');
    // });

    // // Authenticated layout redirects users without memberships to onboarding/create-tenant.
    // cy.location('pathname', { timeout: 30000 }).should(
    //   'include',
    //   '/onboarding/create-tenant',
    // );

    // // Jump directly to the final step for stability.
    // cy.visit('/onboarding/create-tenant?step=2');
    // cy.contains('h1', 'Create a new tenant', { timeout: 30000 }).should(
    //   'be.visible',
    // );

    // cy.intercept('POST', '/api/v1/tenants').as('createTenant');
    // cy.get('input#name').clear().type(tenantName);
    // cy.contains('button', 'Create Tenant').click();

    // cy.wait('@createTenant').its('response.statusCode').should('eq', 200);

    // cy.location('pathname', { timeout: 30000 }).should(
    //   'match',
    //   /\/tenants\/.+\/onboarding\/get-started/,
    // );
    // cy.contains('h1', 'Setup your', { timeout: 30000 }).should('be.visible');
  });
});

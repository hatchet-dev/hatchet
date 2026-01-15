describe('Onboarding: learn workflow', () => {
  it('completes steps 1-4', () => {
    const ts = Date.now();
    const tenantName = `CypressLearnWorkflow${String(ts).slice(-6)}`;

    cy.login('owner');
    cy.visit('/');

    // Create a fresh tenant so we always land on /overview with a clean onboarding flow.
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .click();
    cy.get('[data-cy="new-tenant"]').click();

    cy.location('pathname', { timeout: 5000 }).should(
      'include',
      '/onboarding/create-tenant',
    );
    cy.get('input#name').filter(':visible').first().clear().type(tenantName);
    cy.intercept('POST', '/api/v1/tenants').as('createTenant');
    cy.contains('button', 'Create Tenant').click();
    cy.wait('@createTenant').its('response.statusCode').should('eq', 200);

    cy.location('pathname', { timeout: 30000 }).should(
      'match',
      /\/tenants\/.+\/overview/,
    );

    // Step 1: Install the CLI
    cy.contains('h2', 'Setup your local environment').should('be.visible');
    cy.contains('hatchet --version').should('be.visible');
    cy.contains('[role="tab"]', '2 Set your profile').click();

    // Step 2: Set your profile (generate + copy token to enable Continue)
    cy.contains('Add a Hatchet CLI profile using an API token.').should(
      'be.visible',
    );
    cy.intercept('POST', '/api/v1/tenants/*/api-tokens').as('createApiToken');
    cy.contains('button', 'Generate token for this command')
      .should('be.enabled')
      .click();
    cy.wait('@createApiToken').its('response.statusCode').should('eq', 200);

    cy.contains('hatchet profile add').should('be.visible');
    cy.contains('hatchet profile add')
      .parents('div.relative')
      .first()
      .within(() => {
        // Copy button has no accessible label; scope to this block and click the first button.
        cy.get('button').filter(':visible').first().click();
      });

    cy.contains('[role="tab"]', '3 Project quickstart').click();

    // Step 3: Project quickstart
    cy.contains('hatchet quickstart').should('be.visible');
    cy.contains('[role="tab"]', '4 Run a task').click();

    // Step 4: Run a task
    cy.contains('hatchet trigger').should('be.visible');
    cy.get('[role="tabpanel"]')
      .filter(':visible')
      .first()
      .within(() => {
        cy.get('button')
          .filter(':visible')
          .then(($buttons) => {
            const withText = [...$buttons].filter(
              (button) => button.innerText.trim().length > 0,
            );
            const finish =
              withText.find((button) => /finish/i.test(button.innerText)) ||
              withText[withText.length - 1];

            cy.wrap(finish).click();
          });
      });

    cy.location('pathname', { timeout: 30000 }).should(
      'match',
      /\/tenants\/.+\/runs/,
    );
  });
});

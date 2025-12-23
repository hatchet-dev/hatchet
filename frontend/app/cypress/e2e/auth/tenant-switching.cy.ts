describe('Tenants: switching', () => {
  beforeEach(() => {
    cy.login('owner');
    cy.visit('/');
    cy.clearAllLocalStorage();
  });

  it('should survive page reloads and root redirect', () => {
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .click();

    cy.get('div[data-value="tenant1"]').filter(':visible').first().click();

    cy.get('button[aria-label="Select a tenant"]')
      .should('have.text', 'Tenant 1')
      .filter(':visible')
      .click();

    cy.get('div[data-value="tenant2"]').filter(':visible').first().click();

    cy.get('button[aria-label="Select a tenant"]').should(
      'have.text',
      'Tenant 2',
    );
    cy.reload();
    cy.get('button[aria-label="Select a tenant"]').should(
      'have.text',
      'Tenant 2',
    );
    cy.visit('/');
    cy.get('button[aria-label="Select a tenant"]').should(
      'have.text',
      'Tenant 2',
    );
  });

  it('should not break on login with different user', () => {
    cy.login('owner');
    cy.visit('/');
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .click();
    cy.get('div[data-value="tenant1"]').filter(':visible').first().click();
    cy.get('button[aria-label="Select a tenant"]')
      .should('have.text', 'Tenant 1')
      .filter(':visible')
      .click();
    cy.get('div[data-value="tenant2"]').filter(':visible').first().click();
    cy.get('button[aria-label="Select a tenant"]').should(
      'have.text',
      'Tenant 2',
    );
    cy.get('button[aria-label="User Menu"]')
      .filter(':visible')
      .should('be.visible')
      .first()
      .click();
    // Menu item includes a keyboard shortcut, so match by substring.
    cy.contains('[role="menuitem"]', 'Log out').filter(':visible').click();

    cy.login('member');
    cy.visit('/');
    cy.get('button[aria-label="Select a tenant"]').should(
      'have.text',
      'Tenant 1',
    );
  });
});

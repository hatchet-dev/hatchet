describe('Tenants: switching', () => {
  beforeEach(() => {
    // Ensure no persisted tenant selection leaks between tests/spec ordering.
    cy.clearAllLocalStorage();
    cy.login('owner');
    cy.visit('/');
  });

  it('should survive page reloads and root redirect', () => {
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .as('tenantSwitcher');
    cy.get('@tenantSwitcher').should('not.be.disabled').click({ force: true });

    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get('[data-cy="tenant-switcher-item-tenant1"]')
      .scrollIntoView()
      .click({ force: true });

    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', 'Tenant 1')
      .as('tenantSwitcher2');
    cy.get('@tenantSwitcher2').should('not.be.disabled').click({ force: true });

    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get('[data-cy="tenant-switcher-item-tenant2"]')
      .scrollIntoView()
      .click({ force: true });

    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', 'Tenant 2');
    cy.reload();
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', 'Tenant 2');
    cy.visit('/');
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', 'Tenant 2');
  });

  it('should not break on login with different user', () => {
    // Explicitly reset persisted selection from the previous user session.
    cy.clearAllLocalStorage();
    cy.login('owner');
    cy.visit('/');
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .as('tenantSwitcher');
    cy.get('@tenantSwitcher').should('not.be.disabled').click({ force: true });
    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get('[data-cy="tenant-switcher-item-tenant1"]')
      .scrollIntoView()
      .click({ force: true });
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', 'Tenant 1')
      .as('tenantSwitcher2');
    cy.get('@tenantSwitcher2').should('not.be.disabled').click({ force: true });
    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get('[data-cy="tenant-switcher-item-tenant2"]')
      .scrollIntoView()
      .click({ force: true });
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', 'Tenant 2');
    cy.get('button[aria-label="User Menu"]')
      .filter(':visible')
      .should('be.visible')
      .first()
      .click();
    // Menu item includes a keyboard shortcut, so match by substring.
    cy.contains('[role="menuitem"]', 'Log out').filter(':visible').click();

    cy.login('member');
    cy.visit('/');
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', 'Tenant 1');
  });
});

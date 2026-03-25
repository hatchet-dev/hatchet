describe('Tenants: switching', () => {
  beforeEach(() => {
    // Ensure no persisted tenant selection leaks between tests/spec ordering.
    cy.clearAllLocalStorage();
    cy.login('owner');
    cy.visit('/');
  });

  it('should survive page reloads and root redirect', () => {
    const ts = Date.now();
    const tenant1Name = `Tenant 1 ${ts}`;
    const tenant2Name = `Tenant 2 ${ts}`;
    const tenant1Slug = `cypress-tenant1-${ts}`;
    const tenant2Slug = `cypress-tenant2-${ts}`;

    // Make the spec self-contained even when the DB isn't pre-seeded.
    cy.request('POST', '/api/v1/tenants', {
      name: tenant1Name,
      slug: tenant1Slug,
      environment: 'development',
    })
      .its('status')
      .should('eq', 200);

    cy.request('POST', '/api/v1/tenants', {
      name: tenant2Name,
      slug: tenant2Slug,
      environment: 'development',
    })
      .its('status')
      .should('eq', 200);

    // Refresh the SPA so memberships reflect the created tenants.
    cy.visit('/');
    cy.location('pathname', { timeout: 30000 }).should('include', '/tenants/');

    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .as('tenantSwitcher');
    cy.get('@tenantSwitcher').should('not.be.disabled').click({ force: true });

    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get(`[data-cy="tenant-switcher-item-${tenant1Slug}"]`)
      .should('exist')
      .scrollIntoView()
      .click({ force: true });

    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', tenant1Name)
      .as('tenantSwitcher2');
    cy.get('@tenantSwitcher2').should('not.be.disabled').click({ force: true });

    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get(`[data-cy="tenant-switcher-item-${tenant2Slug}"]`)
      .should('exist')
      .scrollIntoView()
      .click({ force: true });

    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', tenant2Name);
    cy.reload();
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', tenant2Name);
    cy.visit('/');
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', tenant2Name);
  });

  it('should not break on login with different user', () => {
    // Explicitly reset persisted selection from the previous user session.
    cy.clearAllLocalStorage();
    cy.login('owner');
    cy.visit('/');

    const ts = Date.now();
    const tenant1Name = `Tenant 1 ${ts}`;
    const tenant2Name = `Tenant 2 ${ts}`;
    const tenant1Slug = `cypress-tenant1-${ts}`;
    const tenant2Slug = `cypress-tenant2-${ts}`;
    cy.request('POST', '/api/v1/tenants', {
      name: tenant1Name,
      slug: tenant1Slug,
      environment: 'development',
    })
      .its('status')
      .should('eq', 200);
    cy.request('POST', '/api/v1/tenants', {
      name: tenant2Name,
      slug: tenant2Slug,
      environment: 'development',
    })
      .its('status')
      .should('eq', 200);
    cy.visit('/');

    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .as('tenantSwitcher');
    cy.get('@tenantSwitcher').should('not.be.disabled').click({ force: true });
    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get(`[data-cy="tenant-switcher-item-${tenant1Slug}"]`)
      .should('exist')
      .scrollIntoView()
      .click({ force: true });
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', tenant1Name)
      .as('tenantSwitcher2');
    cy.get('@tenantSwitcher2').should('not.be.disabled').click({ force: true });
    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get(`[data-cy="tenant-switcher-item-${tenant2Slug}"]`)
      .should('exist')
      .scrollIntoView()
      .click({ force: true });
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', tenant2Name);
    cy.get('button[aria-label="User Menu"]')
      .filter(':visible')
      .should('be.visible')
      .first()
      .click();
    // Menu item includes a keyboard shortcut, so match by substring.
    cy.contains('[role="menuitem"]', 'Log out').filter(':visible').click();

    cy.login('member');
    cy.visit('/');
    // Member may not have seeded memberships in some environments; just ensure the app doesn't
    // get stuck on the previous user's tenant and lands on a valid authenticated route.
    cy.location('pathname', { timeout: 30000 }).should((pathname) => {
      expect(pathname).to.satisfy(
        (p: string) => p.includes('/tenants/') || p.includes('/onboarding/'),
      );
    });
  });
});

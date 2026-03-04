import { seededUsers } from '../../support/seeded-users.generated';

describe('Create Tenant: redirect to invites', () => {
  it('should redirect to invites page when user has pending invites', () => {
    const ts = Date.now();
    const tenantName = `InviteRedirectTenant${ts}`;
    const tenantSlug = `invite-redirect-tenant-${ts}`;

    // Step 1: Login as owner and create a tenant
    cy.visit('/auth/login');
    cy.get('input#email').type(seededUsers.owner.email);
    cy.get('input#password').type(seededUsers.owner.password);
    cy.get('form')
      .filter(':visible')
      .first()
      .within(() => {
        cy.contains('button', /^Sign In$/)
          .should('be.enabled')
          .click();
      });
    cy.location('pathname', { timeout: 30000 }).should(
      'match',
      /\/tenants\/.+/,
    );

    // Create a new tenant for the invite
    cy.request({
      method: 'POST',
      url: '/api/v1/tenants',
      body: {
        name: tenantName,
        slug: tenantSlug,
        environment: 'development',
      },
    })
      .its('status')
      .should('eq', 200);

    // Refresh to get the new tenant
    cy.visit('/');
    cy.location('pathname', { timeout: 30000 }).should(
      'match',
      /\/tenants\/.+/,
    );

    // Switch to the new tenant
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .click({ force: true });
    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get(`[data-cy="tenant-switcher-item-${tenantSlug}"]`)
      .should('exist')
      .scrollIntoView()
      .click({ force: true });

    // Get tenant ID from URL
    cy.location('pathname', { timeout: 30000 })
      .should('match', /\/tenants\/([^/]+)/)
      .then((pathname) => {
        const match = pathname.match(/\/tenants\/([^/]+)/);
        const tenantId = match![1];

        // Step 2: Create an invite for the member user
        cy.request({
          method: 'POST',
          url: `/api/v1/tenants/${tenantId}/invites`,
          body: {
            email: seededUsers.member.email,
            role: 'MEMBER',
          },
        }).then((response) => {
          expect(response.status).to.eq(201);
        });
      });

    // Step 3: Logout
    cy.get('button[aria-label="User Menu"]')
      .filter(':visible')
      .should('be.visible')
      .first()
      .click();
    cy.contains('[role="menuitem"]', 'Log out').filter(':visible').click();
    cy.location('pathname').should('include', '/auth/login');

    // Step 4: Login as member (who has pending invite)
    cy.get('input#email').type(seededUsers.member.email);
    cy.get('input#password').type(seededUsers.member.password);
    cy.get('form')
      .filter(':visible')
      .first()
      .within(() => {
        cy.contains('button', /^Sign In$/)
          .should('be.enabled')
          .click();
      });

    // Wait for the navigation after sign in to complete
    cy.location('pathname', { timeout: 30000 }).should('not.eq', '/auth/login');

    // Step 5: Try to navigate to create-tenant page
    // The user should be redirected to invites page
    cy.visit('/onboarding/create-tenant', { failOnStatusCode: false });

    // Step 6: Verify redirect to invites page
    cy.location('pathname', { timeout: 10000 }).should(
      'eq',
      '/onboarding/invites',
    );

    // Verify the invite is displayed
    cy.contains(`You got an invitation to join ${tenantName}`).should(
      'be.visible',
    );

    // Step 7: Accept the invite to clean up (prevent affecting other tests)
    cy.intercept('POST', '/api/v1/users/invites/accept').as('acceptInvite');
    cy.contains(`You got an invitation to join ${tenantName}`)
      .parent()
      .contains('button', 'Accept')
      .should('be.visible')
      .click();

    cy.wait('@acceptInvite').its('response.statusCode').should('eq', 200);

    // Verify redirect to tenant page
    cy.location('pathname', { timeout: 10000 }).should(
      'match',
      /\/tenants\/[^/]+/,
    );
  });
});

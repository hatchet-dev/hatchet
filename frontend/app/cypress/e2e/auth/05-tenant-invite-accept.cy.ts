import { seededUsers } from '../../support/seeded-users.generated';

describe('Tenant Invite: accept', () => {
  let tenant2Id: string;

  it('should redirect to tenant page after accepting invite', () => {
    const ts = Date.now();
    const tenant2Name = `Tenant 2 ${ts}`;
    const tenant2Slug = `cypress-tenant2-${ts}`;

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

    // Ensure Tenant 2 exists even if the DB isn't pre-seeded.
    cy.request({
      method: 'POST',
      url: '/api/v1/tenants',
      body: {
        name: tenant2Name,
        slug: tenant2Slug,
        environment: 'development',
      },
    })
      .its('status')
      .should('eq', 200);
    // Refresh so the tenant switcher sees the updated memberships list.
    cy.visit('/');
    cy.location('pathname', { timeout: 30000 }).should(
      'match',
      /\/tenants\/.+/,
    );
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .should('exist');

    // Switch to Tenant 2
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .click({ force: true });
    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get(`[data-cy="tenant-switcher-item-${tenant2Slug}"]`)
      .should('exist')
      .scrollIntoView()
      .click({ force: true });

    // Get the Tenant 2 ID from the URL
    cy.location('pathname', { timeout: 30000 })
      .should('match', /\/tenants\/([^/]+)/)
      .then((pathname) => {
        const match = pathname.match(/\/tenants\/([^/]+)/);
        tenant2Id = match![1];
        cy.wrap(tenant2Id).as('tenant2Id');

        // Create a tenant invite for the member user to join Tenant 2
        cy.request({
          method: 'POST',
          url: `/api/v1/tenants/${tenant2Id}/invites`,
          body: {
            email: 'member@example.com',
            role: 'MEMBER',
          },
        }).then((response) => {
          expect(response.status).to.eq(201);
        });
      });

    cy.get('button[aria-label="User Menu"]')
      .filter(':visible')
      .should('be.visible')
      .first()
      .click();
    cy.contains('[role="menuitem"]', 'Log out').filter(':visible').click();

    cy.location('pathname').should('include', '/auth/login');
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
    cy.location('pathname', { timeout: 5000 }).should(
      'eq',
      '/onboarding/invites',
    );

    // Find the specific invite and accept it
    cy.contains(`You got an invitation to join ${tenant2Name}`).should(
      'be.visible',
    );

    // Step 4: Accept the invite - register intercept before clicking
    cy.intercept('POST', '/api/v1/users/invites/accept').as('acceptInvite');
    cy.contains(`You got an invitation to join ${tenant2Name}`)
      .parent()
      .contains('button', 'Accept')
      .should('be.visible')
      .click();

    // Wait for the accept API call to complete
    cy.wait('@acceptInvite').its('response.statusCode').should('eq', 200);

    // Step 5: Verify redirect to the tenant page (no infinite loop)
    cy.location('pathname', { timeout: 5000 }).should(
      'match',
      /\/tenants\/[^/]+/,
    );

    // Verify we're on Tenant 2
    cy.get('@tenant2Id').then((id) => {
      cy.location('pathname').should('include', `/tenants/${id}`);
    });

    // Verify the tenant switcher shows Tenant 2
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', tenant2Name);
  });
});

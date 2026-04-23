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

    cy.get('button[aria-label="Open account menu"]')
      .filter(':visible')
      .first()
      .click();
    cy.contains('Log out').click();

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
    cy.location('pathname', { timeout: 30000 }).should(
      'match',
      /\/tenants\/.+/,
    );

    // Open the notification dropdown and click the tenant invite notification
    cy.get('[data-cy="notifications-button"]', { timeout: 10000 })
      .filter(':visible')
      .first()
      .click();
    cy.contains(`Tenant invite: ${tenant2Name}`).first().click();

    // Should be on the invites page now
    cy.location('pathname', { timeout: 5000 }).should(
      'eq',
      '/onboarding/invites',
    );

    // Find the specific invite and accept it
    cy.contains(`invited to join the ${tenant2Name} tenant`).should('exist');

    // Accept the invite
    cy.intercept('POST', '/api/v1/users/invites/accept').as('acceptInvite');
    cy.contains(`invited to join the ${tenant2Name} tenant`)
      .parent()
      .contains('button', 'Accept')
      .should('exist')
      .click();

    cy.wait('@acceptInvite').its('response.statusCode').should('eq', 200);

    // Wait for the accepted invite card to be removed from the DOM before
    // looking for remaining Decline buttons. The app awaits a user-universe
    // refetch before removing the card, so there's a window where the stale
    // Decline button is still visible.
    cy.contains(`invited to join the ${tenant2Name} tenant`).should(
      'not.exist',
    );

    // Decline all remaining invites so the page redirects
    const declineAll = (remaining = 20) => {
      cy.get('body').then(($body) => {
        if (
          remaining > 0 &&
          $body.find('button:contains("Decline")').length > 0
        ) {
          cy.intercept('POST', '/api/v1/users/invites/reject').as(
            'rejectInvite',
          );
          cy.contains('button', 'Decline').click({ force: true });
          cy.wait('@rejectInvite');
          declineAll(remaining - 1);
        }
      });
    };
    declineAll();

    // Verify redirect to the tenant page
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

import { seededUsers } from '../../support/seeded-users.generated';

describe('Tenant Invite: decline', () => {
  it('should close the invite modal after declining all invites', () => {
    const ts = Date.now();
    const tenantName = `DeclineTenant${ts}`;
    const tenantSlug = `decline-tenant-${ts}`;

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

    cy.get('button[aria-label="Open account menu"]')
      .filter(':visible')
      .first()
      .click();
    cy.contains('Log out').click();
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

    // Wait for navigation after sign in
    cy.location('pathname', { timeout: 30000 }).should(
      'match',
      /\/tenants\/.+/,
    );

    // The invite modal auto-opens when the member has pending invites.
    cy.get('[role="dialog"]', { timeout: 10000 }).should('be.visible');

    // Verify the invite is displayed in the modal
    cy.contains(tenantName).should('be.visible');

    // Step 5: Decline all invites
    const declineAll = (remaining = 20) => {
      if (remaining === 0) {
        return;
      }
      // Use cy.document() (no retry) so we read the live DOM state at this exact
      // moment without Cypress retrying for 15 s when the dialog has closed.
      cy.document().then((doc) => {
        const btn = doc.querySelector(
          '[role="dialog"][data-state="open"] button[aria-label="Decline"]',
        );
        if (!btn) {
          return;
        }
        cy.intercept('POST', '/api/v1/users/invites/reject').as('rejectInvite');
        // Also intercept the invites refetch that invalidatePendingInvites()
        // triggers so we can wait for React to re-render before recursing.
        cy.intercept('GET', '/api/v1/users/invites*').as('invitesRefetch');
        // Click via a requeryable cy.get() chain: a refetch settling re-renders
        // the modal, and a raw element captured above can detach from the DOM
        // before the click lands (cy.wrap() cannot requery a detached node).
        cy.get(
          '[role="dialog"][data-state="open"] button[aria-label="Decline"]',
        )
          .first()
          .click({ force: true });
        cy.wait('@rejectInvite').its('response.statusCode').should('eq', 200);
        cy.wait('@invitesRefetch');
        declineAll(remaining - 1);
      });
    };
    declineAll();

    // Step 6: Verify the modal closes after all invites are declined
    cy.get('[role="dialog"]', { timeout: 10000 }).should('not.exist');

    // Should remain on a tenant page (modal closes in-place; no redirect)
    cy.location('pathname', { timeout: 10000 }).should(
      'match',
      /\/tenants\/.+/,
    );
  });
});

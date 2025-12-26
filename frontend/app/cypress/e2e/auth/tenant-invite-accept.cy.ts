import { seededUsers } from '../../support/seeded-users.generated';

describe('Tenant Invite: accept', () => {
  let inviteId: string;
  let tenant2Id: string;

  beforeEach(() => {
    cy.clearAllLocalStorage();
  });

  it('should redirect to tenant page after accepting invite', () => {
    // Step 1: Login as owner and switch to Tenant 2
    cy.visit('/');
    cy.wait(500);
    cy.get('input#email').type(seededUsers.owner.email);
    cy.wait(300);
    cy.get('input#password').type(seededUsers.owner.password);
    cy.wait(300);
    cy.get('form')
      .filter(':visible')
      .first()
      .within(() => {
        cy.contains('button', /^Sign In$/)
          .should('be.enabled')
          .click();
      });
    cy.location('pathname', { timeout: 30000 }).should('match', /\/tenants\/.+/);
    cy.wait(1000);
    cy.get('button[aria-label="Select a tenant"]').filter(':visible').should('exist');

    // Switch to Tenant 2
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .click({ force: true });
    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get('[data-cy="tenant-switcher-item-tenant2"]')
      .scrollIntoView()
      .click({ force: true });

    // Get the Tenant 2 ID from the URL
    cy.location('pathname', { timeout: 30000 })
      .should('match', /\/tenants\/([^/]+)/)
      .then((pathname) => {
        const match = pathname.match(/\/tenants\/([^/]+)/);
        tenant2Id = match![1];

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
          inviteId = response.body.metadata.id;
        });
      });

    // Step 2: Logout and login as the member user who received the invite
    cy.wait(500);
    cy.get('button[aria-label="User Menu"]')
      .filter(':visible')
      .should('be.visible')
      .first()
      .click();
    cy.wait(500);
    cy.contains('[role="menuitem"]', 'Log out').filter(':visible').click();
    cy.wait(1000);

    // Step 3: Login as member and visit root - should auto-redirect to invites page
    cy.visit('/');
    cy.intercept('POST', '/api/v1/users/login').as('memberLogin');
    cy.get('input#email').type(seededUsers.member.email);
    cy.wait(300);
    cy.get('input#password').type(seededUsers.member.password);
    cy.wait(300);
    cy.get('form')
      .filter(':visible')
      .first()
      .within(() => {
        cy.contains('button', /^Sign In$/)
          .should('be.enabled')
          .click();
      });
    cy.wait('@memberLogin').its('response.statusCode').should('eq', 200);
    cy.wait(1000);
    cy.location('pathname', { timeout: 5000 }).should(
      'eq',
      '/onboarding/invites',
    );

    // Verify exactly one invite is displayed
    cy.contains('You got an invitation to join Tenant 2').should('be.visible');
    cy.get('button').contains('Accept').should('have.length', 1);

    // Step 4: Accept the invite
    cy.intercept('POST', `/api/v1/tenant-invites/${inviteId}/accept`).as(
      'acceptInvite',
    );
    cy.contains('button', 'Accept').click();

    // Wait for the accept API call to complete
    cy.wait('@acceptInvite').its('response.statusCode').should('eq', 200);

    // Step 5: Verify redirect to the tenant page (no infinite loop)
    cy.location('pathname', { timeout: 5000 }).should(
      'match',
      /\/tenants\/[^/]+/,
    );

    // Clear inviteId so afterEach doesn't try to reject it
    inviteId = '';

    // Verify we're on Tenant 2
    cy.location('pathname').should('include', `/tenants/${tenant2Id}`);

    // Verify the tenant switcher shows Tenant 2
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', 'Tenant 2');
  });

  it('should redirect to invites page when navigating to root with pending invite', () => {
    // Step 1: Login as owner and switch to Tenant 2
    cy.visit('/');
    cy.wait(500);
    cy.get('input#email').type(seededUsers.owner.email);
    cy.wait(300);
    cy.get('input#password').type(seededUsers.owner.password);
    cy.wait(300);
    cy.get('form')
      .filter(':visible')
      .first()
      .within(() => {
        cy.contains('button', /^Sign In$/)
          .should('be.enabled')
          .click();
      });
    cy.location('pathname', { timeout: 30000 }).should('match', /\/tenants\/.+/);
    cy.wait(1000);
    cy.get('button[aria-label="Select a tenant"]').filter(':visible').should('exist');

    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .click({ force: true });
    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get('[data-cy="tenant-switcher-item-tenant2"]')
      .scrollIntoView()
      .click({ force: true });

    cy.location('pathname', { timeout: 30000 })
      .should('match', /\/tenants\/([^/]+)/)
      .then((pathname) => {
        const match = pathname.match(/\/tenants\/([^/]+)/);
        tenant2Id = match![1];

        cy.request({
          method: 'POST',
          url: `/api/v1/tenants/${tenant2Id}/invites`,
          body: {
            email: 'member@example.com',
            role: 'MEMBER',
          },
        }).then((response) => {
          expect(response.status).to.eq(201);
          inviteId = response.body.metadata.id;
        });
      });

    // Step 2: Logout and login as member
    cy.wait(500);
    cy.get('button[aria-label="User Menu"]')
      .filter(':visible')
      .first()
      .click();
    cy.wait(500);
    cy.contains('[role="menuitem"]', 'Log out').filter(':visible').click();
    cy.wait(1000);

    cy.visit('/');
    cy.intercept('POST', '/api/v1/users/login').as('memberLogin');
    cy.get('input#email').type(seededUsers.member.email);
    cy.wait(300);
    cy.get('input#password').type(seededUsers.member.password);
    cy.wait(300);
    cy.get('form')
      .filter(':visible')
      .first()
      .within(() => {
        cy.contains('button', /^Sign In$/)
          .should('be.enabled')
          .click();
      });
    cy.wait('@memberLogin').its('response.statusCode').should('eq', 200);
    cy.wait(1000);

    // Step 3: Navigate to root - should redirect to invites page
    cy.visit('/');
    cy.location('pathname', { timeout: 5000 }).should(
      'eq',
      '/onboarding/invites',
    );

    // Verify exactly one invite is displayed
    cy.contains('You got an invitation to join Tenant 2').should('be.visible');
    cy.get('button').contains('Accept').should('have.length', 1);

    // Step 4: Navigate to a tenant page, then back to root
    cy.visit('/tenants/tenant1/workflows');
    cy.location('pathname').should('include', '/workflows');

    // Navigate back to root - should redirect to invites again
    cy.visit('/');
    cy.location('pathname', { timeout: 5000 }).should(
      'eq',
      '/onboarding/invites',
    );

    // Invite should still be visible
    cy.contains('You got an invitation to join Tenant 2').should('be.visible');
    cy.get('button').contains('Accept').should('have.length', 1);
  });

  it('should handle accepting invite when already on invites page', () => {
    // Step 1: Login as owner and switch to Tenant 2
    cy.visit('/');
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
    cy.location('pathname', { timeout: 30000 }).should('match', /\/tenants\/.+/);
    cy.get('button[aria-label="Select a tenant"]').filter(':visible').should('exist');

    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .click({ force: true });
    cy.get('[data-cy="tenant-switcher-list"]').should('be.visible');
    cy.get('[data-cy="tenant-switcher-item-tenant2"]')
      .scrollIntoView()
      .click({ force: true });

    cy.location('pathname', { timeout: 30000 })
      .should('match', /\/tenants\/([^/]+)/)
      .then((pathname) => {
        const match = pathname.match(/\/tenants\/([^/]+)/);
        tenant2Id = match![1];

        cy.request({
          method: 'POST',
          url: `/api/v1/tenants/${tenant2Id}/invites`,
          body: {
            email: 'member@example.com',
            role: 'MEMBER',
          },
        }).then((response) => {
          expect(response.status).to.eq(201);
          inviteId = response.body.metadata.id;
        });
      });

    // Step 2: Logout and login as member
    cy.wait(500);
    cy.get('button[aria-label="User Menu"]')
      .filter(':visible')
      .first()
      .click();
    cy.wait(500);
    cy.contains('[role="menuitem"]', 'Log out').filter(':visible').click();
    cy.wait(1000);

    cy.visit('/');
    cy.intercept('POST', '/api/v1/users/login').as('memberLogin');
    cy.get('input#email').type(seededUsers.member.email);
    cy.wait(300);
    cy.get('input#password').type(seededUsers.member.password);
    cy.wait(300);
    cy.get('form')
      .filter(':visible')
      .first()
      .within(() => {
        cy.contains('button', /^Sign In$/)
          .should('be.enabled')
          .click();
      });
    cy.wait('@memberLogin').its('response.statusCode').should('eq', 200);
    cy.wait(1000);

    // Step 3: Directly navigate to invites page
    cy.visit('/onboarding/invites');
    cy.location('pathname').should('eq', '/onboarding/invites');

    // Verify exactly one invite is displayed
    cy.contains('You got an invitation to join Tenant 2').should('be.visible');
    cy.get('button').contains('Accept').should('have.length', 1);

    // Step 4: Accept the invite
    cy.intercept('POST', `/api/v1/tenant-invites/${inviteId}/accept`).as(
      'acceptInvite',
    );
    cy.contains('button', 'Accept').click();
    cy.wait('@acceptInvite').its('response.statusCode').should('eq', 200);

    // Step 5: Verify no infinite redirect loop - should land on tenant page
    cy.location('pathname', { timeout: 5000 }).should(
      'match',
      /\/tenants\/[^/]+/,
    );

    // Verify we're on Tenant 2
    cy.location('pathname').should('include', `/tenants/${tenant2Id}`);

    // Should not redirect back to invites
    cy.location('pathname', { timeout: 2000 }).should(
      'not.include',
      '/onboarding/invites',
    );

    // Verify the tenant switcher shows Tenant 2
    cy.get('button[aria-label="Select a tenant"]')
      .filter(':visible')
      .first()
      .should('contain.text', 'Tenant 2');

    // Clear inviteId so afterEach doesn't try to reject it
    inviteId = '';
  });
});

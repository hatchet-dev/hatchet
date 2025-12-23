import { seededUsers } from '../../support/seeded-users.generated';

describe('auth: login', () => {
  it('should login a user with username and password', () => {
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
    cy.location('pathname').should('include', '/tenants/');
    cy.get('button[aria-label="User Menu"]').filter(':visible').first().click();
    // `data-cy="user-name"` exists in both the trigger and the dropdown content; scope to the open menu.
    cy.get('[role="menu"]')
      .filter(':visible')
      .first()
      .within(() => {
        cy.get('[data-cy="user-name"]')
          .filter(':visible')
          .first()
          .should('have.text', seededUsers.owner.name);
      });
  });
});

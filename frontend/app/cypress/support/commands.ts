import { loginSession } from './flows/auth';
import { seededUsers } from './seeded-users.generated';

declare global {
  // Cypress exposes its typings via a global namespace; augmenting it is the intended pattern here.
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace Cypress {
    interface Chainable {
      /**
       * Log in (cached via `cy.session()` + `cacheAcrossSpecs`) using a seeded user.
       */
      login(user: keyof typeof seededUsers): Chainable<null>;
    }
  }
}

Cypress.Commands.add('login', (user: keyof typeof seededUsers) => {
  return loginSession(user);
});

export {};

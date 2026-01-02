// Cypress support file. Load custom commands here.
import './commands';

// TanStack Router throws `Response` objects for route loaders (e.g. 403/404) which
// are handled by the app's error UI, but Cypress can still treat them as uncaught.
// Ignore ONLY this specific case so real application errors still fail tests.
Cypress.on('uncaught:exception', (err) => {
  if (
    typeof err?.message === 'string' &&
    err.message.includes('[object Response]')
  ) {
    return false;
  }

  return true;
});

export {};

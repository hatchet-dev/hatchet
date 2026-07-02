// Cypress support file. Load custom commands here.
import './commands';

// TanStack Router throws `Response` objects for route loaders (e.g. 403/404) which
// are handled by the app's error UI, but Cypress can still treat them as uncaught.
// Ignore ONLY this specific case so real application errors still fail tests.
Cypress.on('uncaught:exception', (err) => {
  if (typeof err?.message === 'string') {
    // TanStack Router throws `Response` objects for route loaders (handled by app error UI).
    if (err.message.includes('[object Response]')) {
      return false;
    }
    // React Query cancels in-flight requests when components unmount during navigation.
    // CancelledError is expected and not a real failure.
    if (
      err.message === 'CancelledError' ||
      err.message.includes('CancelledError')
    ) {
      return false;
    }
  }
  // React Query may throw CancelledError objects directly (not wrapped in Error).
  if (err?.constructor?.name === 'CancelledError') {
    return false;
  }

  return true;
});

export {};

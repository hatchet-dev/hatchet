/**
 * Custom sequencer to run e2e tests in deterministic alphabetical order.
 * Jest's default discovery order can differ between macOS and Linux.
 */
const Sequencer = require('@jest/test-sequencer').default;

class E2ESequencer extends Sequencer {
  sort(tests) {
    return [...tests].sort((a, b) => a.path.localeCompare(b.path));
  }
}

module.exports = E2ESequencer;

import {
  availableUseCases,
  isLanguageSupported,
  resolveLanguage,
  scaffoldCommand,
  triggerCommand,
  workerDevCommand,
} from './use-case-options';
import assert from 'node:assert/strict';
import { test } from 'node:test';

test('scaffoldCommand derives the CLI command from the selection', () => {
  assert.equal(
    scaffoldCommand({ useCase: 'simple', language: 'python' }),
    'hatchet quickstart --use-case simple --language python',
  );
  assert.equal(
    scaffoldCommand({ useCase: 'simple', language: 'typescript' }),
    'hatchet quickstart --use-case simple --language typescript',
  );
  assert.equal(
    scaffoldCommand({ useCase: 'simple', language: 'go' }),
    'hatchet quickstart --use-case simple --language go',
  );
  assert.equal(
    scaffoldCommand({ useCase: 'scheduled', language: 'python' }),
    'hatchet quickstart --use-case scheduled --language python',
  );
  assert.equal(
    scaffoldCommand({ useCase: 'scheduled', language: 'typescript' }),
    'hatchet quickstart --use-case scheduled --language typescript',
  );
  assert.equal(
    scaffoldCommand({ useCase: 'scheduled', language: 'go' }),
    'hatchet quickstart --use-case scheduled --language go',
  );
});

test('triggerCommand uses the trigger name registered by each template', () => {
  assert.equal(
    triggerCommand('simple', 'Test'),
    'hatchet trigger simple --profile "Test"',
  );
  assert.equal(
    triggerCommand('scheduled', 'Test'),
    'hatchet trigger manual-run --profile "Test"',
  );
});

test('worker and trigger commands quote and escape the profile name', () => {
  assert.equal(
    workerDevCommand('Acme "prod" $team'),
    'hatchet worker dev --profile "Acme \\"prod\\" \\$team"',
  );
  assert.equal(
    triggerCommand('simple', 'Acme "prod" $team'),
    'hatchet trigger simple --profile "Acme \\"prod\\" \\$team"',
  );
});

test('language compatibility matches the published templates', () => {
  for (const useCase of ['simple', 'scheduled'] as const) {
    assert.equal(isLanguageSupported(useCase, 'python'), true);
    assert.equal(isLanguageSupported(useCase, 'typescript'), true);
    assert.equal(isLanguageSupported(useCase, 'go'), true);
    assert.equal(resolveLanguage(useCase, 'typescript'), 'typescript');
  }
});

test('only the shippable use cases are selectable', () => {
  assert.deepEqual(Object.keys(availableUseCases), ['simple', 'scheduled']);

  // The @ts-expect-error assertions are checked by tsc; the wrapper is
  // never executed.
  const rejectedByTypes = () => {
    // @ts-expect-error roadmap use cases are not selectable
    triggerCommand('pdf', 'Test');
    // @ts-expect-error roadmap use cases are not selectable
    scaffoldCommand({ useCase: 'claudeAgent', language: 'go' });
  };
  void rejectedByTypes;
});

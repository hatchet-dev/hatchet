import { WorkerStatus } from '../../../../../lib/api/generated/data-contracts';
import {
  applyLanguageChange,
  applyTabChange,
  applyUseCaseChange,
  defaultOnboardingState,
  hasQualifiedWorker,
  normalizeOnboardingState,
  onboardingStorageKey,
  qualifiedRunQueryParams,
  type OnboardingPersistedState,
} from './onboarding-state';
import assert from 'node:assert/strict';
import { test } from 'node:test';

const CONFIRMED_AT = '2026-07-26T18:00:00.000Z';

const storedState = (
  overrides: Partial<OnboardingPersistedState> = {},
): OnboardingPersistedState => ({
  useCase: 'simple',
  language: 'python',
  tab: 'runTask',
  hidden: false,
  selectionConfirmedAt: CONFIRMED_AT,
  ...overrides,
});

test('storage keys are tenant-scoped', () => {
  const keyA = onboardingStorageKey('tenant-a');
  const keyB = onboardingStorageKey('tenant-b');

  assert.notEqual(keyA, keyB);
  assert.equal(keyA, 'hatchet:onboarding:tenant-a');
});

test('restart writes the complete default state', () => {
  // The Settings restart control persists exactly this object.
  assert.deepEqual(defaultOnboardingState(), {
    useCase: 'simple',
    language: 'python',
    tab: 'chooseUseCase',
    hidden: false,
    selectionConfirmedAt: null,
  });
});

test('malformed state falls back to the defaults', () => {
  const fallback = defaultOnboardingState();

  assert.deepEqual(normalizeOnboardingState(null), fallback);
  assert.deepEqual(normalizeOnboardingState('not-an-object'), fallback);
  assert.deepEqual(normalizeOnboardingState(42), fallback);
  assert.deepEqual(normalizeOnboardingState({}), fallback);
});

test('a valid stored state survives normalization unchanged', () => {
  const state = normalizeOnboardingState(storedState());

  assert.deepEqual(state, storedState());
});

test('a stored use case no longer in the catalog resets the selection', () => {
  const state = normalizeOnboardingState(
    storedState({ useCase: 'pdf' as never, language: 'go' }),
  );

  assert.equal(state.useCase, 'simple');
  assert.equal(state.tab, 'chooseUseCase');
  assert.equal(state.selectionConfirmedAt, null);
});

test('an unknown stored language resets the selection', () => {
  const state = normalizeOnboardingState(
    storedState({ language: 'rust' as never, hidden: true }),
  );

  // The fallback language is compatible with the use case, so the reset
  // comes from the unknown value itself.
  assert.equal(state.language, 'python');
  assert.equal(state.tab, 'chooseUseCase');
  assert.equal(state.selectionConfirmedAt, null);
  assert.equal(state.hidden, true);
});

test('an unknown tab falls back to Choose use case', () => {
  const state = normalizeOnboardingState(
    storedState({ tab: 'removedTab' as never }),
  );

  assert.equal(state.tab, 'chooseUseCase');
});

test('the confirmation timestamp survives only as a valid date-time', () => {
  assert.equal(
    normalizeOnboardingState(storedState()).selectionConfirmedAt,
    CONFIRMED_AT,
  );
  assert.equal(
    normalizeOnboardingState(
      storedState({ selectionConfirmedAt: 'not-a-date' }),
    ).selectionConfirmedAt,
    null,
  );
});

test('completion is never part of the persisted state', () => {
  const state = normalizeOnboardingState({
    ...storedState(),
    completed: true,
  });

  assert.deepEqual(Object.keys(state).sort(), [
    'hidden',
    'language',
    'selectionConfirmedAt',
    'tab',
    'useCase',
  ]);
});

test('changing use case clears the confirmation timestamp and preserves the language', () => {
  const next = applyUseCaseChange(storedState(), 'scheduled');

  assert.equal(next.useCase, 'scheduled');
  assert.equal(next.language, 'python');
  assert.equal(next.selectionConfirmedAt, null);
  // The user is re-choosing, so the current tab stays where it is.
  assert.equal(next.tab, 'runTask');
});

test('re-selecting the current use case changes nothing', () => {
  const state = storedState();

  assert.equal(applyUseCaseChange(state, 'simple'), state);
});

test('changing language clears the confirmation timestamp', () => {
  const next = applyLanguageChange(storedState(), 'go');

  assert.equal(next.language, 'go');
  assert.equal(next.selectionConfirmedAt, null);
});

test('re-selecting the current language changes nothing', () => {
  const state = storedState();

  assert.equal(applyLanguageChange(state, 'python'), state);
});

test('leaving Choose use case records the confirmation timestamp when none exists', () => {
  const now = '2026-07-27T09:00:00.000Z';
  const choosing = storedState({
    tab: 'chooseUseCase',
    selectionConfirmedAt: null,
  });

  const toInstall = applyTabChange(choosing, 'install', now);
  assert.equal(toInstall.tab, 'install');
  assert.equal(toInstall.selectionConfirmedAt, now);

  const toQuickstart = applyTabChange(choosing, 'quickstart', now);
  assert.equal(toQuickstart.tab, 'quickstart');
  assert.equal(toQuickstart.selectionConfirmedAt, now);
});

test('tab navigation preserves an existing confirmation timestamp', () => {
  const now = '2026-07-27T09:00:00.000Z';

  const later = applyTabChange(
    storedState({ tab: 'quickstart' }),
    'runTask',
    now,
  );
  assert.equal(later.selectionConfirmedAt, CONFIRMED_AT);

  // Only a use-case or language change clears the timestamp.
  const back = applyTabChange(storedState(), 'chooseUseCase', now);
  assert.equal(back.tab, 'chooseUseCase');
  assert.equal(back.selectionConfirmedAt, CONFIRMED_AT);
});

test('re-selecting the current tab changes nothing', () => {
  const state = storedState();

  assert.equal(
    applyTabChange(state, 'runTask', '2026-07-27T09:00:00.000Z'),
    state,
  );
});

test('the run query filters to completed runs after the confirmed selection', () => {
  const params = qualifiedRunQueryParams(CONFIRMED_AT);

  assert.equal(params.since, CONFIRMED_AT);
  assert.deepEqual(params.statuses, ['COMPLETED']);
  assert.equal(params.limit, 1);
  assert.equal(params.only_tasks, false);
});

const worker = (status: WorkerStatus, createdAt: string) => ({
  status,
  metadata: { id: 'w', createdAt, updatedAt: createdAt },
});

const BEFORE_CONFIRMATION = '2026-07-26T17:00:00.000Z';
const AFTER_CONFIRMATION = '2026-07-26T19:00:00.000Z';

test('a worker registered before the confirmed selection does not qualify', () => {
  // The previous selection's worker is still listed and ACTIVE;
  // registration time is what excludes it.
  assert.equal(
    hasQualifiedWorker(
      [worker(WorkerStatus.ACTIVE, BEFORE_CONFIRMATION)],
      CONFIRMED_AT,
    ),
    false,
  );
});

test('an active worker registered after the confirmed selection qualifies', () => {
  assert.equal(
    hasQualifiedWorker(
      [worker(WorkerStatus.ACTIVE, AFTER_CONFIRMATION)],
      CONFIRMED_AT,
    ),
    true,
  );

  // Registration exactly at the confirmation timestamp counts.
  assert.equal(
    hasQualifiedWorker(
      [worker(WorkerStatus.ACTIVE, CONFIRMED_AT)],
      CONFIRMED_AT,
    ),
    true,
  );
});

test('a disconnected or paused worker does not qualify', () => {
  assert.equal(
    hasQualifiedWorker(
      [worker(WorkerStatus.INACTIVE, AFTER_CONFIRMATION)],
      CONFIRMED_AT,
    ),
    false,
  );
  assert.equal(
    hasQualifiedWorker(
      [worker(WorkerStatus.PAUSED, AFTER_CONFIRMATION)],
      CONFIRMED_AT,
    ),
    false,
  );
});

test('no confirmed selection means no worker qualifies', () => {
  // Selection changes clear the timestamp (the transition tests above),
  // so this rule is what resets worker qualification with them.
  assert.equal(
    hasQualifiedWorker([worker(WorkerStatus.ACTIVE, AFTER_CONFIRMATION)], null),
    false,
  );
});

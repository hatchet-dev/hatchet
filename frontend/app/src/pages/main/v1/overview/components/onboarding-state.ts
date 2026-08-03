import {
  V1TaskStatus,
  WorkerStatus,
  type Worker,
} from '../../../../../lib/api/generated/data-contracts';
import {
  workflowLanguageOptions,
  workflowStepOptions,
  type WorkflowLanguageKey,
  type WorkflowStepKey,
} from './onboarding-options';
import {
  availableUseCases,
  resolveLanguage,
  type AvailableUseCaseKey,
} from './use-case-options';

// Onboarding state for one tenant, persisted to localStorage. Completion
// is deliberately absent. It is derived from the runs query, so clearing
// storage can never fabricate or lose it.
export type OnboardingPersistedState = {
  useCase: AvailableUseCaseKey;
  language: WorkflowLanguageKey;
  tab: WorkflowStepKey;
  // True after Skip or Finish. The Overview then renders no onboarding;
  // recovery is Restart onboarding on the tenant General settings page.
  hidden: boolean;
  // ISO date-time recorded when the use-case and language selection is
  // confirmed by navigating past Choose use case. Workers and runs qualify
  // only from this moment on, so nothing left over from a previously
  // selected use case or language can satisfy the current one.
  selectionConfirmedAt: string | null;
};

// Onboarding state is a property of the tenant being onboarded, so the key
// embeds the tenant id.
export function onboardingStorageKey(tenantId: string): string {
  return `hatchet:onboarding:${tenantId}`;
}

export function defaultOnboardingState(): OnboardingPersistedState {
  return {
    useCase: 'simple',
    language: workflowLanguageOptions.python.value,
    tab: workflowStepOptions.chooseUseCase.value,
    hidden: false,
    selectionConfirmedAt: null,
  };
}

// Validates a value read from storage, falling back field by field for
// anything unknown or malformed. A stored use case that is no longer
// selectable, or a stored language its use case does not support,
// invalidates the whole selection. The tab returns to Choose use case and
// selectionConfirmedAt clears, because downstream progress belonged to a
// selection that no longer exists. The hidden flag survives repairs;
// hiding is not selection-dependent.
export function normalizeOnboardingState(
  value: unknown,
): OnboardingPersistedState {
  const fallback = defaultOnboardingState();

  if (typeof value !== 'object' || value === null) {
    return fallback;
  }

  const raw = value as Record<string, unknown>;

  const useCaseValid =
    typeof raw.useCase === 'string' && raw.useCase in availableUseCases;
  const useCase = useCaseValid
    ? (raw.useCase as AvailableUseCaseKey)
    : fallback.useCase;

  const languageKnown =
    typeof raw.language === 'string' && raw.language in workflowLanguageOptions;
  const languageCandidate = languageKnown
    ? (raw.language as WorkflowLanguageKey)
    : fallback.language;
  const language = resolveLanguage(useCase, languageCandidate);
  // languageKnown is checked separately so an unknown stored language
  // invalidates the selection even when the fallback is compatible.
  const selectionValid =
    useCaseValid && languageKnown && language === languageCandidate;

  const tab =
    selectionValid &&
    typeof raw.tab === 'string' &&
    raw.tab in workflowStepOptions
      ? (raw.tab as WorkflowStepKey)
      : fallback.tab;

  const selectionConfirmedAt =
    selectionValid &&
    typeof raw.selectionConfirmedAt === 'string' &&
    !Number.isNaN(Date.parse(raw.selectionConfirmedAt))
      ? raw.selectionConfirmedAt
      : null;

  return {
    useCase,
    language,
    tab,
    hidden: raw.hidden === true,
    selectionConfirmedAt,
  };
}

export function applyUseCaseChange(
  state: OnboardingPersistedState,
  nextUseCase: AvailableUseCaseKey,
): OnboardingPersistedState {
  if (nextUseCase === state.useCase) {
    return state;
  }

  return {
    ...state,
    useCase: nextUseCase,
    language: resolveLanguage(nextUseCase, state.language),
    selectionConfirmedAt: null,
  };
}

export function applyLanguageChange(
  state: OnboardingPersistedState,
  nextLanguage: WorkflowLanguageKey,
): OnboardingPersistedState {
  const language = resolveLanguage(state.useCase, nextLanguage);
  if (language === state.language) {
    return state;
  }

  return {
    ...state,
    language,
    selectionConfirmedAt: null,
  };
}

// Every tab is directly clickable, so moving to any tab past Choose use
// case confirms the selection when no timestamp exists yet; otherwise a
// direct click on a later tab would leave worker and run detection
// disabled. An existing timestamp survives all navigation, including
// returning to Choose use case. The timestamp is a parameter so the
// transition stays deterministic.
export function applyTabChange(
  state: OnboardingPersistedState,
  nextTab: WorkflowStepKey,
  confirmationTimestamp: string,
): OnboardingPersistedState {
  if (nextTab === state.tab) {
    return state;
  }

  const confirmsSelection =
    state.selectionConfirmedAt === null &&
    nextTab !== workflowStepOptions.chooseUseCase.value;

  return {
    ...state,
    tab: nextTab,
    selectionConfirmedAt: confirmsSelection
      ? confirmationTimestamp
      : state.selectionConfirmedAt,
  };
}

// A worker qualifies when it is currently ACTIVE and it registered at or
// after the confirmed selection, so a worker left over from a previously
// selected use case or language cannot satisfy the current one. The
// workers list returns every worker with a heartbeat in the last 24
// hours, and the API computes status from heartbeat staleness with a 5
// second threshold, so a killed worker can read ACTIVE for up to that
// long plus one poll interval.
export function hasQualifiedWorker(
  workers: Array<Pick<Worker, 'status' | 'metadata'>>,
  selectionConfirmedAt: string | null,
): boolean {
  if (!selectionConfirmedAt) {
    return false;
  }

  const confirmedAtMs = Date.parse(selectionConfirmedAt);

  return workers.some(
    (worker) =>
      worker.status === WorkerStatus.ACTIVE &&
      Date.parse(worker.metadata.createdAt) >= confirmedAtMs,
  );
}

// Query parameters for detecting the onboarding run. The server compares
// against `since`, so a run created before the confirmed selection can
// never qualify, and one row is enough to answer whether onboarding is
// complete.
export function qualifiedRunQueryParams(selectionConfirmedAt: string) {
  return {
    offset: 0,
    limit: 1,
    statuses: [V1TaskStatus.COMPLETED],
    since: selectionConfirmedAt,
    only_tasks: false,
    include_payloads: false,
  };
}

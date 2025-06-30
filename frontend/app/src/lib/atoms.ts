import { atom } from 'jotai';
import { TenantBillingState } from './api/generated/cloud/data-contracts';

const getInitialValue = <T>(key: string, defaultValue?: T): T | undefined => {
  const item = localStorage.getItem(key);

  if (item !== null) {
    return JSON.parse(item) as T;
  }

  if (defaultValue !== undefined) {
    return defaultValue;
  }

  return;
};

type Plan = 'free' | 'starter' | 'growth';

export type BillingContext = {
  state: TenantBillingState | undefined;
  setPollBilling: (pollBilling: boolean) => void;
  plan: Plan;
  hasPaymentMethods: boolean;
};

const lastTimeRange = 'lastTimeRange';

const lastTimeRangeAtomInit = atom(
  getInitialValue<string>(lastTimeRange, '1h'),
);

export const lastTimeRangeAtom = atom(
  (get) => get(lastTimeRangeAtomInit),
  (_get, set, newVal: string) => {
    set(lastTimeRangeAtomInit, newVal);
    localStorage.setItem(lastTimeRange, JSON.stringify(newVal));
  },
);

const lastWorkerMetricsTimeRange = 'lastWorkerMetricsTimeRange';

const lastWorkerMetricsTimeRangeAtomInit = atom(
  getInitialValue<string>(lastWorkerMetricsTimeRange, '1h'),
);

export const lastWorkerMetricsTimeRangeAtom = atom(
  (get) => get(lastWorkerMetricsTimeRangeAtomInit),
  (_get, set, newVal: string) => {
    set(lastWorkerMetricsTimeRangeAtomInit, newVal);
    localStorage.setItem(lastWorkerMetricsTimeRange, JSON.stringify(newVal));
  },
);

export type ViewOptions = 'graph' | 'minimap';

const preferredWorkflowRunViewKey = 'wrView';

const preferredWorkflowRunViewAtomInit = atom(
  getInitialValue<ViewOptions>(preferredWorkflowRunViewKey, 'minimap'),
);

export const preferredWorkflowRunViewAtom = atom(
  (get) => get(preferredWorkflowRunViewAtomInit),
  (_get, set, newVal: ViewOptions) => {
    set(preferredWorkflowRunViewAtomInit, newVal);
    localStorage.setItem(preferredWorkflowRunViewKey, JSON.stringify(newVal));
  },
);

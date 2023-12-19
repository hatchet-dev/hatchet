import { atom } from 'jotai';
import { Tenant } from './api';

const getInitialValue = <T>(key: string): T | undefined => {
  const item = localStorage.getItem(key);

  if (item !== null) {
    return JSON.parse(item) as T;
  }

  return;
};

const currTenantKey = 'currTenant';

const currTenantAtomInit = atom(getInitialValue<Tenant>(currTenantKey));

export const currTenantAtom = atom(
  (get) => get(currTenantAtomInit),
  (_get, set, newVal: Tenant) => {
    set(currTenantAtomInit, newVal);
    localStorage.setItem(currTenantKey, JSON.stringify(newVal));
  },
);

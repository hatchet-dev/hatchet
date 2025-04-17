// import { atom, useAtom } from 'jotai';
// import { Tenant, queries } from './api';
// import { useSearchParams } from 'react-router-dom';
// import { useCallback, useEffect, useMemo, useState } from 'react';
// import { useQuery } from '@tanstack/react-query';
// import { TenantBillingState } from './api/generated/cloud/data-contracts';
// import { Evaluate } from './can/shared/permission.base';
// import useUser from '@/next/hooks/use-user';
// import useApiMeta from '@/next/hooks/use-api-meta';
// const getInitialValue = <T>(key: string, defaultValue?: T): T | undefined => {
//   const item = localStorage.getItem(key);

//   if (item !== null) {
//     return JSON.parse(item) as T;
//   }

//   if (defaultValue !== undefined) {
//     return defaultValue;
//   }

//   return;
// };

// const lastTenantKey = 'lastTenant';

// const lastTenantAtomInit = atom(getInitialValue<Tenant>(lastTenantKey));

// export const lastTenantAtom = atom(
//   (get) => get(lastTenantAtomInit),
//   (_get, set, newVal: Tenant) => {
//     set(lastTenantAtomInit, newVal);
//     localStorage.setItem(lastTenantKey, JSON.stringify(newVal));
//   },
// );

// type Plan = 'free' | 'starter' | 'growth';

// export type BillingContext = {
//   state: TenantBillingState | undefined;
//   setPollBilling: (pollBilling: boolean) => void;
//   plan: Plan;
//   hasPaymentMethods: boolean;
// };

// type Can = (evalFn: Evaluate) => ReturnType<Evaluate>;

// type TenantContextPresent = {
//   tenant: Tenant;
//   tenantId: string;
//   setTenant: (tenant: Tenant) => void;
//   billing: BillingContext;
//   can: Can;
// };

// type TenantContextMissing = {
//   tenant: undefined;
//   tenantId: undefined;
//   setTenant: (tenant: Tenant) => void;
//   billing: undefined;
//   can: Can;
// };

// type TenantContext = TenantContextPresent | TenantContextMissing;

// // search param sets the tenant, the last tenant set is used if the search param is empty,
// // otherwise the first membership is used
// export function useTenantOLD(): TenantContext {
//   const user = useUser();

//   const [lastTenant, setLastTenant] = useAtom(lastTenantAtom);
//   const [searchParams, setSearchParams] = useSearchParams();

//   const setTenant = useCallback(
//     (tenant: Tenant) => {
//       const newSearchParams = new URLSearchParams(searchParams);
//       newSearchParams.set('tenant', tenant.metadata.id);
//       setSearchParams(newSearchParams, { replace: true });
//       setLastTenant(tenant);
//     },
//     [searchParams, setSearchParams, setLastTenant],
//   );

//   const memberships = useMemo(() => user?.memberships || [], [user]);

//   const findTenant = useCallback(
//     (tenantId: string) => {
//       return memberships?.find((m) => m.tenant?.metadata.id === tenantId)
//         ?.tenant;
//     },
//     [memberships],
//   );

//   const computedCurrTenant = useMemo(() => {
//     const currTenantId = searchParams.get('tenant') || undefined;
//     const lastTenantId = lastTenant?.metadata.id || undefined;

//     // If the current tenant is set as a query param, use it
//     if (currTenantId) {
//       const tenant = findTenant(currTenantId);

//       if (tenant) {
//         return tenant;
//       }
//     }

//     // Otherwise, if a tenant was set in Jotai, use that as a fallback
//     if (lastTenantId) {
//       const tenant = findTenant(lastTenantId);

//       if (tenant) {
//         return tenant;
//       }
//     }

//     return;
//   }, [searchParams, findTenant, lastTenant]);

//   const currTenantId = searchParams.get('tenant');
//   const currTenant = currTenantId ? findTenant(currTenantId) : undefined;

//   const tenant = currTenant || computedCurrTenant;

//   // If the tenant is not set as a query param at any point,
//   // set it.
//   // NOTE: This is helpful mostly for debugging to easily grab
//   // the tenant from the URL.
//   useEffect(() => {
//     const currentTenantParam = searchParams.get('tenant');
//     if (!currentTenantParam && tenant) {
//       setTenant(tenant);
//     }
//   }, [searchParams, tenant, setTenant]);

//   // Set the correct path for tenant version
//   // NOTE: this is hacky and not ideal
//   // Tenant Billing State

//   const [pollBilling, setPollBilling] = useState(false);

//   const meta = useApiMeta();

//   const billingState = useQuery({
//     ...queries.cloud.billing(tenant?.metadata?.id || ''),
//     enabled: tenant && !!meta.cloud?.canBill,
//     refetchInterval: pollBilling ? 1000 : false,
//   });

//   const subscriptionPlan: Plan = useMemo(() => {
//     const plan = billingState.data?.subscription?.plan;
//     if (!plan) {
//       return 'free';
//     }
//     return plan as Plan;
//   }, [billingState.data?.subscription?.plan]);

//   const hasPaymentMethods = useMemo(() => {
//     return (billingState.data?.paymentMethods?.length || 0) > 0;
//   }, [billingState.data?.paymentMethods]);

//   const billingContext: BillingContext = useMemo(() => {
//     return {
//       state: billingState.data,
//       setPollBilling,
//       plan: subscriptionPlan,
//       hasPaymentMethods,
//     };
//   }, [billingState.data, setPollBilling, subscriptionPlan, hasPaymentMethods]);

//   const can = useCallback(
//     (evalFn: Evaluate) => {
//       return evalFn({
//         tenant,
//         billing: billingContext,
//         meta: meta.cloud,
//       });
//     },
//     [billingContext, meta.cloud, tenant],
//   );

//   if (!tenant) {
//     return {
//       tenant: undefined,
//       tenantId: undefined,
//       setTenant,
//       billing: undefined,
//       can,
//     };
//   }

//   return {
//     tenant,
//     tenantId: tenant.metadata.id,
//     setTenant,
//     billing: billingContext,
//     can,
//   };
// }

// const lastTimeRange = 'lastTimeRange';

// const lastTimeRangeAtomInit = atom(
//   getInitialValue<string>(lastTimeRange, '1h'),
// );

// export const lastTimeRangeAtom = atom(
//   (get) => get(lastTimeRangeAtomInit),
//   (_get, set, newVal: string) => {
//     set(lastTimeRangeAtomInit, newVal);
//     localStorage.setItem(lastTimeRange, JSON.stringify(newVal));
//   },
// );

// const lastWorkerMetricsTimeRange = 'lastWorkerMetricsTimeRange';

// const lastWorkerMetricsTimeRangeAtomInit = atom(
//   getInitialValue<string>(lastWorkerMetricsTimeRange, '1h'),
// );

// export const lastWorkerMetricsTimeRangeAtom = atom(
//   (get) => get(lastWorkerMetricsTimeRangeAtomInit),
//   (_get, set, newVal: string) => {
//     set(lastWorkerMetricsTimeRangeAtomInit, newVal);
//     localStorage.setItem(lastWorkerMetricsTimeRange, JSON.stringify(newVal));
//   },
// );

// export type ViewOptions = 'graph' | 'minimap';

// const preferredWorkflowRunViewKey = 'wrView';

// const preferredWorkflowRunViewAtomInit = atom(
//   getInitialValue<ViewOptions>(preferredWorkflowRunViewKey, 'minimap'),
// );

// export const preferredWorkflowRunViewAtom = atom(
//   (get) => get(preferredWorkflowRunViewAtomInit),
//   (_get, set, newVal: ViewOptions) => {
//     set(preferredWorkflowRunViewAtomInit, newVal);
//     localStorage.setItem(preferredWorkflowRunViewKey, JSON.stringify(newVal));
//   },
// );

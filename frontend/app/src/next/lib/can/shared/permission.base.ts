import { BillingHook } from '@/next/hooks/use-billing';
import { Tenant, TenantMember, User } from '@/lib/api/generated/data-contracts';
import useApiMeta from '@/next/hooks/use-api-meta';
export type EvaluateContext = {
  user?: User;
  membership?: TenantMember['role'];
  tenant?: Tenant;
  billing?: BillingHook['billing'];
  meta?: ReturnType<typeof useApiMeta>;
};

export type EvaluateResult = {
  allowed: boolean;
  rejectReason?: RejectReason;
  message?: string;
};

export type Evaluate = (context: EvaluateContext) => EvaluateResult;

export type PermissionSet<K = any> = Record<string, (resource?: K) => Evaluate>;

export enum RejectReason {
  BILLING_REQUIRED = 'BILLING_REQUIRED',
  UPGRADE_REQUIRED = 'UPGRADE_REQUIRED',
  ROLE_REQUIRED = 'ROLE_REQUIRED',
  CLOUD_ONLY = 'CLOUD_ONLY',
}

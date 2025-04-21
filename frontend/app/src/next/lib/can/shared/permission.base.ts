import { BillingHook } from '@/next/hooks/use-billing';
import { APICloudMetadata } from '@/lib/api/generated/cloud/data-contracts';
import { Tenant, TenantMember, User } from '@/lib/api/generated/data-contracts';

export type EvaluateContext = {
  user?: User;
  membership?: TenantMember['role'];
  tenant?: Tenant;
  billing?: BillingHook['data'];
  meta?: APICloudMetadata;
};

export type EvaluateResult = {
  allowed: boolean;
  reason?: RejectReason;
  message?: string;
};

export type Evaluate = (context: EvaluateContext) => EvaluateResult;

export type PermissionSet<K = any> = Record<string, (resource?: K) => Evaluate>;

export enum RejectReason {
  BILLING_REQUIRED = 'BILLING_REQUIRED',
  UPGRADE_REQUIRED = 'UPGRADE_REQUIRED',
  ROLE_REQUIRED = 'ROLE_REQUIRED',
}

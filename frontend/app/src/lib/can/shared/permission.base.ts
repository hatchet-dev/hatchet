import { APICloudMetadata } from '@/lib/api/generated/cloud/data-contracts';
import { Tenant } from '@/lib/api/generated/data-contracts';
import { BillingContext } from '@/lib/atoms';

export type EvaluateContext = {
  tenant?: Tenant;
  billing?: BillingContext;
  meta?: APICloudMetadata;
};

export type Evaluate = (
  context: EvaluateContext,
) => [boolean, RejectReason | undefined];

export type PermissionSet<K = any> = Record<string, (resource?: K) => Evaluate>;

export enum RejectReason {
  BILLING_REQUIRED = 'BILLING_REQUIRED',
  UPGRADE_REQUIRED = 'UPGRADE_REQUIRED',
  PLAN_RESTRICTION = 'PLAN_RESTRICTION',
}

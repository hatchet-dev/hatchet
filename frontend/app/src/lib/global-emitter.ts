import type { CreateOrganizationInviteRequest } from '@/lib/api/generated/cloud/data-contracts';
import type { TenantInvite } from '@/lib/api/generated/data-contracts';
import makeEmitter from 'better-emitter';

export const globalEmitter = makeEmitter<{
  'create-new-tenant': { defaultOrganizationId?: string };
  'create-tenant-invite': { tenantId: string };
  'create-organization-invite': {
    organizationId: string;
    organizationName: string;
  };
  'tenant-invite-created': { tenantId: string; invite: TenantInvite };
  'organization-invite-created': {
    organizationId: string;
    invite: CreateOrganizationInviteRequest;
  };
}>();

import type { CreateOrganizationInviteRequest } from '@/lib/api/generated/cloud/data-contracts';
import type { TenantInvite } from '@/lib/api/generated/data-contracts';
import makeEmitter from 'better-emitter';

export const globalEmitter = makeEmitter<{
  'create-new-tenant': { defaultOrganizationId?: string; allTenantTags?: string[] };
  'create-tenant-invite': { tenantId: string };
  'add-org-member-to-tenant': { tenantId: string; organizationId: string };
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

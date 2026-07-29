import type { TenantInvite } from '@/lib/api/generated/data-contracts';
import type { OrganizationInviteCreateRequest } from '@/lib/api/organization-wrapper';
import makeEmitter from 'better-emitter';

export const globalEmitter = makeEmitter<{
  'create-new-tenant': {
    defaultOrganizationId?: string;
    allTenantTags?: string[];
  };
  'create-tenant-invite': {
    tenantId?: string;
    organizationId?: string;
    defaultEmail?: string;
  };
  'create-organization-invite': {
    organizationId: string;
    organizationName: string;
  };
  'tenant-invite-created': { tenantId: string; invite: TenantInvite };
  'organization-invite-created': {
    organizationId: string;
    invite: OrganizationInviteCreateRequest;
  };
  'open-invite-modal': Record<string, never>;
}>();

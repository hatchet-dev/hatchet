import { appRoutes } from '@/router';

export type SettingsNavItem = {
  key: string;
  name: string;
  to: string;
  params: Record<string, string>;
  exact?: boolean;
};

export type SettingsNavGroup = {
  key: string;
  title: string;
  items: SettingsNavItem[];
};

export function settingsNavGroups(opts: {
  tenantId?: string;
  orgId?: string;
  canBill?: boolean;
  isControlPlaneEnabled?: boolean;
  isOrganizationOwner?: boolean;
  canManageSso?: boolean;
}): SettingsNavGroup[] {
  const groups: SettingsNavGroup[] = [];

  if (opts.tenantId) {
    const params = { tenant: opts.tenantId };

    groups.push({
      key: 'tenant',
      title: 'Tenant',
      items: [
        {
          key: 'general',
          name: 'General',
          to: appRoutes.tenantSettingsOverviewRoute.to,
          params,
        },
        {
          key: 'members',
          name: 'Members',
          to: appRoutes.tenantSettingsMembersRoute.to,
          params,
        },
        {
          key: 'api-tokens',
          name: 'API Tokens',
          to: appRoutes.tenantSettingsApiTokensRoute.to,
          params,
        },
        {
          key: 'integrations',
          name: 'Integrations',
          to: appRoutes.tenantSettingsIntegrationsRoute.to,
          params,
        },
        {
          key: 'resource-limits',
          name: 'Resource Limits',
          to: appRoutes.tenantSettingsResourceLimitsRoute.to,
          params,
        },
      ],
    });
  }

  if (opts.isControlPlaneEnabled && opts.orgId) {
    const params = { organization: opts.orgId };

    groups.push({
      key: 'organization',
      title: 'Organization',
      items: [
        {
          key: 'organization-general',
          name: 'General',
          to: appRoutes.organizationsIndexRoute.to,
          params,
          exact: true,
        },
        {
          key: 'organization-team',
          name: 'Team',
          to: appRoutes.organizationTeamRoute.to,
          params,
        },
        {
          key: 'organization-tenants',
          name: 'Tenants',
          to: appRoutes.organizationTenantsRoute.to,
          params,
        },
        {
          key: 'organization-tokens',
          name: 'Management Tokens',
          to: appRoutes.organizationTokensRoute.to,
          params,
        },
        ...(opts.isOrganizationOwner && opts.isControlPlaneEnabled
          ? [
              {
                key: 'organization-regions',
                name: 'Available Regions',
                to: appRoutes.organizationRegionsRoute.to,
                params,
              },
            ]
          : []),
        ...(opts.canManageSso
          ? [
              {
                key: 'organization-sso',
                name: 'SSO',
                to: appRoutes.organizationSsoRoute.to,
                params,
              },
            ]
          : []),
        ...(opts.isControlPlaneEnabled
          ? [
              {
                key: 'organization-audit-log',
                name: 'Audit Log',
                to: appRoutes.organizationAuditLogRoute.to,
                params,
              },
            ]
          : []),
        ...(opts.canBill
          ? [
              {
                key: 'organization-billing',
                name: 'Billing & Usage',
                to: appRoutes.organizationBillingRoute.to,
                params,
              },
            ]
          : []),
      ],
    });
  }

  return groups;
}

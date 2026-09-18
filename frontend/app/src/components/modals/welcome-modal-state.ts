export const WELCOME_KEY = 'hatchet:show-welcome';

// Records, per organization, that the free-plan notice was already shown once
// the tenant started approaching its limits, so it does not reappear on every
// poll or page load.
export const freePlanLimitNoticeKey = (organizationId: string) =>
  `hatchet:free-plan-limit-notice:${organizationId}`;

// Why the free-plan modal is being shown: right after signup (legacy flow), or
// later because the tenant is approaching a free-plan limit.
export type WelcomeReason = 'welcome' | 'approaching-limit';

export const WELCOME_TRIGGER = {
  OrganizationCreated: 'organization_created',
  TenantCreated: 'tenant_created',
} as const;

export type WelcomeTrigger =
  (typeof WELCOME_TRIGGER)[keyof typeof WELCOME_TRIGGER];

export function readWelcomeTrigger(value: string | null) {
  if (value === '1') {
    return WELCOME_TRIGGER.OrganizationCreated;
  }

  if (
    value === WELCOME_TRIGGER.OrganizationCreated ||
    value === WELCOME_TRIGGER.TenantCreated
  ) {
    return value;
  }

  return null;
}

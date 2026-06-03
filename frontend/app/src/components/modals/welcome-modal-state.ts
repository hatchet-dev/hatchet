export const WELCOME_KEY = 'hatchet:show-welcome';

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

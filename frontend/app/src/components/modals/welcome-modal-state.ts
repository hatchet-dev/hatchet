// Records, per organization, that the free-plan notice was already shown, so
// it appears once when the tenant first approaches a limit rather than on
// every poll or page load.
export const freePlanLimitNoticeKey = (organizationId: string) =>
  `hatchet:free-plan-limit-notice:${organizationId}`;

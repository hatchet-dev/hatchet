export const OFFICE_HOURS_URL = 'https://cal.com/team/hatchet/talk-to-us';
export const DISCORD_INVITE_URL = 'https://discord.com/invite/ZMeUafwH89';

export function getCloudCTAUrl(utmCampaign: string) {
  if (import.meta.env.DEV) {
    return 'https://cloud.hatchet.run';
  }

  return `https://cloud.hatchet.run?utm_source=self_hosted_sidebar&utm_medium=app&utm_campaign=${encodeURIComponent(utmCampaign)}`;
}

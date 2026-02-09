import CronPrettifier from 'cronstrue';

export const extractCronTz = (cron: string): string => {
  const tzMatch = cron.match(/^CRON_TZ=([^\s]+)\s+/);
  return tzMatch ? tzMatch[1] : 'UTC';
};

export const formatCron = (cron: string) => {
  const cronExpression = cron.replace(/^CRON_TZ=([^\s]+)\s+/, '');
  return CronPrettifier.toString(cronExpression || '');
};

import { differenceInCalendarDays } from 'date-fns';

export const formatInviteExpiry = (expires: string) => {
  const days = differenceInCalendarDays(new Date(expires), new Date());
  if (days < 0) {
    return '(expired)';
  }
  if (days === 0) {
    return '(expires today)';
  }
  if (days === 1) {
    return '(expires in 1 day)';
  }
  return `(expires in ${days} days)`;
};

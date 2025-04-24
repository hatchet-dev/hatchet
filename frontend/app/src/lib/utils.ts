import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';
import { APIErrors } from './api/generated/data-contracts';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function getFieldErrors(apiErrors: APIErrors): Record<string, string> {
  const fieldErrors: Record<string, string> = {};

  if (!apiErrors.errors) {
    return fieldErrors;
  }

  for (const error of apiErrors.errors) {
    if (error.field && error.description) {
      fieldErrors[error.field] = error.description;
    }
  }

  return fieldErrors;
}

export function capitalize(s: string) {
  if (!s) {
    return '';
  } else if (s.length == 0) {
    return s;
  } else if (s.length == 1) {
    return s.charAt(0).toUpperCase();
  }

  return s.charAt(0).toUpperCase() + s.substring(1).toLowerCase();
}

export function relativeDate(date?: string | number) {
  if (!date) {
    return 'N/A';
  }

  const rtf = new Intl.RelativeTimeFormat('en', {
    localeMatcher: 'best fit', // other values: "lookup"
    numeric: 'auto', // other values: "auto"
    style: 'long', // other values: "short" or "narrow"
  });

  const time = timeFrom(date);
  if (!time) {
    return 'N/A';
  }

  let value = time.time;

  if (time.when === 'past') {
    value = -value;
  }

  return capitalize(rtf.format(value, time.unitOfTime));
}

function timeFrom(time: string | number, secondTime?: string | number) {
  // Get timestamps
  const unixTime = new Date(time).getTime();
  if (!unixTime) {
    return;
  }

  let now = new Date().getTime();

  if (secondTime) {
    now = new Date(secondTime).getTime();
  }

  // Calculate difference
  let difference = unixTime / 1000 - now / 1000;

  // Setup return object
  const tfn: {
    when: 'past' | 'now' | 'future';
    unitOfTime: Intl.RelativeTimeFormatUnit;
    time: number;
  } = {
    when: 'now',
    unitOfTime: 'seconds',
    time: 0,
  };

  // Check if time is in the past, present, or future
  if (difference > 0) {
    tfn.when = 'future';
  } else if (difference < -1) {
    tfn.when = 'past';
  }

  // Convert difference to absolute
  difference = Math.abs(difference);

  // Calculate time unit
  if (difference / (60 * 60 * 24 * 365) > 1) {
    // Years
    tfn.unitOfTime = 'years';
    tfn.time = Math.floor(difference / (60 * 60 * 24 * 365));
  } else if (difference / (60 * 60 * 24 * 45) > 1) {
    // Months
    tfn.unitOfTime = 'months';
    tfn.time = Math.floor(difference / (60 * 60 * 24 * 45));
  } else if (difference / (60 * 60 * 24) > 1) {
    // Days
    tfn.unitOfTime = 'days';
    tfn.time = Math.floor(difference / (60 * 60 * 24));
  } else if (difference / (60 * 60) > 1) {
    // Hours
    tfn.unitOfTime = 'hours';
    tfn.time = Math.floor(difference / (60 * 60));
  } else if (difference / 60 > 1) {
    // Minutes
    tfn.unitOfTime = 'minutes';
    tfn.time = Math.floor(difference / 60);
  } else {
    // Seconds
    tfn.unitOfTime = 'seconds';
    tfn.time = Math.floor(difference);
  }

  // Return time from now data
  return tfn;
}

export function formatDuration(ms: number): string {
  if (ms < 0) {
    return '0s';
  }

  if (ms < 1000) {
    return `${ms}ms`;
  } else if (ms < 60000) {
    return `${(ms / 1000).toFixed(1)}s`;
  } else if (ms < 3600000) {
    return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
  } else {
    const hours = Math.floor(ms / 3600000);
    const minutes = Math.floor((ms % 3600000) / 60000);
    const seconds = Math.floor((ms % 60000) / 1000);
    return `${hours}h ${minutes}m ${seconds}s`;
  }
}

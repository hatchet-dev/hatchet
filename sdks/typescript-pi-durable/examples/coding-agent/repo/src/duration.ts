const UNITS: Record<string, number> = {
  ms: 1,
  s: 1_000,
  m: 60_000,
  h: 3_600_000,
  d: 86_400_000,
};

const PART = /(\d+)(m|ms|s|h|d)/g;

/** Parses a duration like "1h30m" or "500ms" into milliseconds. */
export function parseDuration(input: string): number {
  let total = 0;
  let matched = 0;

  for (const [part, value, unit] of input.trim().matchAll(PART)) {
    total += Number(value) * UNITS[unit];
    matched += part.length;
  }

  if (matched === 0) {
    throw new Error(`invalid duration: "${input}"`);
  }

  return total;
}

/** Formats milliseconds back into the largest whole units, e.g. 5400000 -> "1h30m". */
export function formatDuration(ms: number): string {
  if (ms === 0) return '0ms';
  let rest = ms;
  let out = '';
  for (const unit of ['d', 'h', 'm', 's', 'ms']) {
    const size = UNITS[unit];
    const count = Math.floor(rest / size);
    if (count > 0) {
      out += `${count}${unit}`;
      rest -= count * size;
    }
  }
  return out;
}

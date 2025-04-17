export function percent(value: number, total: number): number {
  const res = Math.round((value / total) * 100);

  if (isNaN(res)) {
    return 0;
  }

  return res;
}

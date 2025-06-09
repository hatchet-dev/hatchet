import { Coupon } from '@/lib/api/generated/cloud/data-contracts';

/**
 * Adjusts a price in cents based on an array of applied coupons.
 * Supports both percent-based and fixed-amount coupons.
 *
 * @param {number} priceCents - The original price in cents.
 * @param {Coupon[] | undefined} coupons - The array of applied coupons.
 * @returns {number} - The adjusted price in cents (rounded to nearest cent).
 */
export function applyCouponsToPrice(
  priceCents: number,
  coupons?: Coupon[],
): number {
  if (!coupons || coupons.length === 0) {
    return priceCents;
  }

  // Apply percent-based coupons first
  let adjusted = priceCents;
  for (const coupon of coupons) {
    if (coupon.percent) {
      adjusted = Math.round(adjusted * (1 - coupon.percent / 100));
    }
  }

  // Then apply fixed-amount coupons
  for (const coupon of coupons) {
    if (coupon.amount_cents) {
      adjusted = Math.max(0, adjusted - coupon.amount_cents);
    }
  }

  return adjusted;
}

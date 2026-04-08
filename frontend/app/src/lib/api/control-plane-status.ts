import { APIControlPlaneMetadata } from '@/lib/api/generated/control-plane/data-contracts';

export function inferControlPlaneEnabled(
  meta: APIControlPlaneMetadata | null | undefined,
): boolean {
  return !!meta;
}

const CONTROL_PLANE_ENABLED_STORAGE_KEY = 'hatchet_control_plane_enabled';

type MetaLikeWithErrors = {
  errors?: unknown;
};

function hasErrorsField(value: unknown): value is MetaLikeWithErrors {
  return !!value && typeof value === 'object' && 'errors' in value;
}

export function inferControlPlaneEnabled(meta: unknown): boolean {
  if (!meta || typeof meta !== 'object') {
    return false;
  }

  return !hasErrorsField(meta) || !meta.errors;
}

export function readStoredControlPlaneEnabled(): boolean | null {
  try {
    const raw = localStorage.getItem(CONTROL_PLANE_ENABLED_STORAGE_KEY);
    if (raw === null) {
      return null;
    }
    return raw === 'true';
  } catch {
    return null;
  }
}

export function writeStoredControlPlaneEnabled(enabled: boolean): void {
  try {
    localStorage.setItem(CONTROL_PLANE_ENABLED_STORAGE_KEY, String(enabled));
  } catch {
    // Ignore storage failures.
  }
}

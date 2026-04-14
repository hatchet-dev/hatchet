import { EditFormValues, FormValues } from './sso-schemas';
import { IdpInfoFromCustomer, ProviderKey, SsoApi } from './sso-types';

/* ============================================================================
 * Utility Functions
 * ========================================================================= */

export function copySsoToClipboard(text: string, onDone?: () => void) {
  if (typeof navigator !== 'undefined' && navigator.clipboard) {
    navigator.clipboard.writeText(text).then(() => onDone?.());
  }
}

export function normalizeSsoApi(api?: Partial<SsoApi>): SsoApi {
  return {
    async get() {
      if (api?.get) {
        return api.get();
      }
      console.error(
        'SsoSetup: api.get not implemented. Pass an `api` prop to implement.',
      );
      return { ok: true, data: {} };
    },
    async upsert(args) {
      if (api?.upsert) {
        return api.upsert(args);
      }
      const message =
        'SsoSetup: api.upsert not implemented. Pass an `api` prop to implement.';
      console.error(message);
      return { ok: false, error: { message } };
    },
    async remove() {
      if (api?.remove) {
        return api.remove();
      }
      const message =
        'SsoSetup: api.remove not implemented. Pass an `api` prop to implement.';
      console.warn(message);
      return { ok: false, error: { message } };
    },
  };
}

/* ============================================================================
 * Data Transformation Functions
 * ========================================================================= */

export function inferSsoProvider(info: IdpInfoFromCustomer): ProviderKey {
  if (info.idpType === 'Okta') {
    return 'Okta';
  }
  if (info.idpType === 'MicrosoftEntra') {
    return 'MicrosoftEntra';
  }
  return 'Generic';
}

export function hydrateSsoForm(
  provider: ProviderKey,
  info: IdpInfoFromCustomer,
): FormValues {
  if (info.idpType === 'Okta') {
    return {
      provider: 'Okta' as const,
      clientId: info.clientId || '',
      clientSecret: info.clientSecret || '',
      ssoDomain: info.ssoDomain || '',
      usesPkce: info.usesPkce ?? true,
    };
  } else if (info.idpType === 'MicrosoftEntra') {
    return {
      provider: 'MicrosoftEntra' as const,
      clientId: info.clientId || '',
      clientSecret: info.clientSecret || '',
      tenantId: info.tenantId || '',
      usesPkce: info.usesPkce ?? true,
    };
  } else {
    // For Generic type, use the provider parameter as fallback or default to "Generic"
    const genericProvider =
      provider === 'Generic' ||
      provider === 'Google' ||
      provider === 'OneLogin' ||
      provider === 'JumpCloud'
        ? provider
        : 'Generic';
    return {
      provider: genericProvider,
      clientId: info.clientId || '',
      clientSecret: info.clientSecret || '',
      authUrl: info.authUrl || '',
      tokenUrl: info.tokenUrl || '',
      userinfoUrl: info.userinfoUrl || '',
      usesPkce: info.usesPkce ?? true,
    };
  }
}

export function toSsoIdpInfo(
  values: FormValues | EditFormValues,
): IdpInfoFromCustomer {
  if (values.provider === 'Okta') {
    return {
      idpType: 'Okta' as const,
      clientId: values.clientId,
      clientSecret: values.clientSecret || undefined, // Send undefined if empty to keep existing
      ssoDomain: values.ssoDomain,
      usesPkce: values.usesPkce,
    } as IdpInfoFromCustomer;
  } else if (values.provider === 'MicrosoftEntra') {
    return {
      idpType: 'MicrosoftEntra' as const,
      clientId: values.clientId,
      clientSecret: values.clientSecret || undefined, // Send undefined if empty to keep existing
      tenantId: values.tenantId,
      usesPkce: values.usesPkce,
    } as IdpInfoFromCustomer;
  } else {
    const v = values as Extract<typeof values, { authUrl: string }>;
    return {
      idpType: 'Generic' as const,
      clientId: v.clientId,
      clientSecret: v.clientSecret || undefined, // Send undefined if empty to keep existing
      authUrl: v.authUrl,
      tokenUrl: v.tokenUrl,
      userinfoUrl: v.userinfoUrl,
      usesPkce: v.usesPkce,
    } as IdpInfoFromCustomer;
  }
}

/* ============================================================================
 * Types & Interfaces
 * ========================================================================= */

export type ProviderKey = "Okta" | "MicrosoftEntra" | "Google" | "OneLogin" | "JumpCloud" | "Generic";

export type IdpInfoFromCustomer =
    | { idpType: "Okta"; clientId: string; clientSecret?: string; ssoDomain: string; usesPkce: boolean }
    | { idpType: "MicrosoftEntra"; clientId: string; clientSecret?: string; tenantId: string; usesPkce: boolean }
    | {
          idpType: "Generic";
          clientId: string;
          clientSecret?: string;
          authUrl: string;
          tokenUrl: string;
          userinfoUrl: string;
          usesPkce: boolean;
      };

export type Result<T> =
    | { ok: true; data: T }
    | { ok: false; error: { message?: string; status?: number; details?: unknown } };

export interface SsoApi {
    get(): Promise<Result<{ idpInfoFromCustomer?: IdpInfoFromCustomer }>>;
    upsert(params: { idpInfoFromCustomer: IdpInfoFromCustomer }): Promise<Result<void>>;
    remove(): Promise<Result<void>>;
}

export interface SsoSetupFormProps {
    redirectUrl: string;
    api?: Partial<SsoApi>;
    onSuccess?: (data: IdpInfoFromCustomer | null) => void;
    initialStep?: 1 | 2;
    isEmbedded?: boolean;
    onFormChange?: (hasChanges: boolean) => void;
}

export interface SsoSetupProps {
    redirectUrl: string;
    api?: Partial<SsoApi>;
}

export enum SsoSetupStep {
    ProviderSelection = 1,
    Configuration = 2,
}

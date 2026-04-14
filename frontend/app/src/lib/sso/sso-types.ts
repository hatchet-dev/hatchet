/* ============================================================================
 * Types & Interfaces
 * ========================================================================= */

export type ProviderKey =
  | 'Okta'
  | 'MicrosoftEntra'
  | 'Google'
  | 'OneLogin'
  | 'JumpCloud'
  | 'Generic';

export type IdpInfoFromCustomer =
  | {
      idpType: 'Okta';
      clientId: string;
      clientSecret?: string;
      ssoDomain: string;
      usesPkce: boolean;
    }
  | {
      idpType: 'MicrosoftEntra';
      clientId: string;
      clientSecret?: string;
      tenantId: string;
      usesPkce: boolean;
    }
  | {
      idpType: 'Generic';
      clientId: string;
      clientSecret?: string;
      authUrl: string;
      tokenUrl: string;
      userinfoUrl: string;
      usesPkce: boolean;
    };

export enum SsoSetupStep {
  ProviderSelection = 1,
  Configuration = 2,
}

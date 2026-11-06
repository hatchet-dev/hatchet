import { controlPlaneApi } from './api';

const path = '/api/v1/control-plane/users/mfa';
export type MFAMethod = {
  id: string;
  kind: 'totp' | 'passkey';
};
export type MFAStatus = {
  pending: boolean;
  totp: boolean;
  passkeys: string[];
  methods: MFAMethod[];
  recoveryRemaining: number;
  canManage: boolean;
};
export const mfaStatus = async () =>
  (await controlPlaneApi.instance.get<MFAStatus>(path)).data;
export const mfaPost = async <T = { recoveryCodes?: string[] }>(
  endpoint: string,
  body: unknown = {},
): Promise<T> =>
  (
    await controlPlaneApi.instance.post<T>(`${path}/${endpoint}`, body, {
      headers: { 'X-Hatchet-MFA': '1', 'Content-Type': 'application/json' },
    })
  ).data;

const decode = (value: string): ArrayBuffer => {
  const raw = atob(value.replace(/-/g, '+').replace(/_/g, '/'));
  return Uint8Array.from(raw, (c) => c.charCodeAt(0)).buffer;
};
const encode = (value: ArrayBuffer): string =>
  btoa(String.fromCharCode(...new Uint8Array(value)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');

type DescriptorJSON = Omit<PublicKeyCredentialDescriptor, 'id'> & {
  id: string;
};
type CreationJSON = Omit<
  PublicKeyCredentialCreationOptions,
  'challenge' | 'user' | 'excludeCredentials'
> & {
  challenge: string;
  user: Omit<PublicKeyCredentialUserEntity, 'id'> & { id: string };
  excludeCredentials?: DescriptorJSON[];
};
type RequestJSON = Omit<
  PublicKeyCredentialRequestOptions,
  'challenge' | 'allowCredentials'
> & {
  challenge: string;
  allowCredentials?: DescriptorJSON[];
};
const descriptor = (item: DescriptorJSON): PublicKeyCredentialDescriptor => ({
  ...item,
  id: decode(item.id),
});

// Closing the browser prompt rejects with NotAllowedError and a link to the
// WebAuthn spec. Callers treat this as a cancellation, not a shown error.
export class PasskeyCancelled extends Error {
  constructor() {
    super('Passkey prompt was closed.');
    this.name = 'PasskeyCancelled';
  }
}

async function requestPasskey(
  request: () => Promise<Credential | null>,
): Promise<PublicKeyCredential | null> {
  try {
    return (await request()) as PublicKeyCredential | null;
  } catch (err) {
    if (
      err instanceof DOMException &&
      (err.name === 'NotAllowedError' || err.name === 'AbortError')
    ) {
      throw new PasskeyCancelled();
    }
    throw err;
  }
}

export async function passkeyCeremony(enroll: boolean) {
  if (!window.PublicKeyCredential || !navigator.credentials) {
    throw new Error(
      'This browser does not support passkeys. Use an authenticator or recovery code.',
    );
  }
  const kind = enroll ? 'passkey-enroll' : 'passkey';
  let credential: PublicKeyCredential | null;
  if (enroll) {
    const { publicKey } = await mfaPost<{ publicKey: CreationJSON }>(
      `${kind}/start`,
    );
    credential = await requestPasskey(() =>
      navigator.credentials.create({
        publicKey: {
          ...publicKey,
          challenge: decode(publicKey.challenge),
          user: { ...publicKey.user, id: decode(publicKey.user.id) },
          excludeCredentials: publicKey.excludeCredentials?.map(descriptor),
        },
      }),
    );
  } else {
    const { publicKey } = await mfaPost<{ publicKey: RequestJSON }>(
      `${kind}/start`,
    );
    credential = await requestPasskey(() =>
      navigator.credentials.get({
        publicKey: {
          ...publicKey,
          challenge: decode(publicKey.challenge),
          allowCredentials: publicKey.allowCredentials?.map(descriptor),
        },
      }),
    );
  }
  if (!credential) {
    throw new Error('Passkey verification was cancelled.');
  }
  const response = credential.response;
  const serialized =
    response instanceof AuthenticatorAttestationResponse
      ? {
          clientDataJSON: encode(response.clientDataJSON),
          attestationObject: encode(response.attestationObject),
          transports: response.getTransports?.() ?? [],
        }
      : (() => {
          const assertion = response as AuthenticatorAssertionResponse;
          return {
            clientDataJSON: encode(assertion.clientDataJSON),
            authenticatorData: encode(assertion.authenticatorData),
            signature: encode(assertion.signature),
            userHandle: assertion.userHandle
              ? encode(assertion.userHandle)
              : null,
          };
        })();
  return mfaPost(`${kind}/finish`, {
    id: credential.id,
    rawId: encode(credential.rawId),
    type: credential.type,
    response: serialized,
    clientExtensionResults: credential.getClientExtensionResults(),
    authenticatorAttachment: credential.authenticatorAttachment,
  });
}

// Package contract holds the constants shared between the serverless operator core and the
// API server: the data ids for secrets encrypted at rest. Both sides must agree on these for
// ciphertext written by one to be readable by the other.
package contract

// SigningSecretEncryptionDataID is the associated data used when encrypting a serverless
// endpoint's signing secret with pkg/encryption. It binds the ciphertext in
// v1_serverless_endpoint.signing_secret_enc to that column so it cannot be replayed as any other
// secret.
const SigningSecretEncryptionDataID = "v1_serverless_endpoint_signing_secret" //nolint:gosec // associated-data label for envelope encryption, not a credential

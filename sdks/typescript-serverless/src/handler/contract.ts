/**
 * The parts of the endpoint contract that are not protobuf messages, mirrored from
 * pkg/serverlessoperator/contract/http.go. The messages themselves are the generated
 * bindings in ../generated/proto.
 */

/** Headers carried by every request the operator sends to an endpoint. */
export const SIGNATURE_HEADER = 'X-Hatchet-Signature';
export const ENDPOINT_ID_HEADER = 'X-Hatchet-Endpoint-Id';
export const TIMESTAMP_HEADER = 'X-Hatchet-Timestamp';

/** Headers carried only by the durable websocket upgrade, which has no body to sign. */
export const NONCE_HEADER = 'X-Hatchet-Nonce';
export const TASK_ID_HEADER = 'X-Hatchet-Task-Id';
export const INVOCATION_HEADER = 'X-Hatchet-Invocation';

/**
 * contract.RequestMaxAge, in seconds: how far a signed timestamp may lie from the endpoint's
 * clock, in either direction, before the request is refused. It applies to the timestamp in
 * every signed POST body and to the upgrade's X-Hatchet-Timestamp.
 */
export const REQUEST_MAX_AGE_SECONDS = 5 * 60;

/** contract.UpgradeMaxAge: RequestMaxAge as it applies to the durable upgrade. */
export const UPGRADE_MAX_AGE_SECONDS = REQUEST_MAX_AGE_SECONDS;

/** contract.TriggerEnvelopeVersion: the `version` field of ServerlessTriggerRequest. */
export const TRIGGER_ENVELOPE_VERSION = 1;

/** contract.DoneStatusEvicted. */
export const DONE_STATUS_EVICTED = 'evicted';

/** Codes of ServerlessErrorFrame. */
export const ERROR_CODE_NON_DETERMINISM = 'nondeterminism';
export const ERROR_CODE_UNSPECIFIED = 'unspecified';

/**
 * The string the durable upgrade signature covers (contract.UpgradeSigningPayload):
 * endpoint_id "." timestamp "." nonce "." task_id "." invocation, each as it appears in its
 * header. The endpoint id is part of it so a signature made for one endpoint cannot be
 * presented to another that shares the signing secret.
 */
export function upgradeSigningPayload(
  endpointId: string,
  timestamp: string,
  nonce: string,
  taskId: string,
  invocation: string
): string {
  return `${endpointId}.${timestamp}.${nonce}.${taskId}.${invocation}`;
}

/**
 * The operator prefixes everything an endpoint registers with `<namespace>_`: workflow
 * names, event keys and the service part of action ids (pkg/serverlessoperator/routing.go).
 * The namespace itself travels without the separator.
 */
export function namespacePrefix(namespace: string): string {
  return namespace ? `${namespace}_` : '';
}

/** Removes the namespace prefix from a workflow name or event key, if present. */
export function stripNamespace(name: string, namespace: string): string {
  const prefix = namespacePrefix(namespace);

  return prefix && name.startsWith(prefix) ? name.slice(prefix.length) : name;
}

/**
 * Removes the namespace prefix from an action id. Only the service part is prefixed
 * (`<ns>_service:verb`), which for a leading prefix is the same as stripping the name.
 */
export function stripActionNamespace(actionId: string, namespace: string): string {
  return stripNamespace(actionId, namespace);
}

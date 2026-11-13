/**
 * Serializes a user payload to JSON for the engine, replacing lone UTF-16 surrogates in
 * string values with U+FFFD, as `TextEncoder` does.
 *
 * A lone surrogate, such as what `text.slice(0, n)` leaves when it cuts through an emoji, is
 * valid in a JavaScript string and `JSON.stringify` writes it as an escape like `\ud83d`. The
 * engine stores payloads as Postgres jsonb, which cannot hold half a surrogate pair, so it
 * rejects them. `JSON.stringify` writes whole pairs as the literal character and a lone
 * surrogate as a lowercase `\udXXX` escape, so a result without `\ud` needs no second pass.
 *
 * Object keys are left as they are: a replacer cannot rename them, so a lone surrogate in a key
 * still reaches the engine and is rejected there.
 *
 * Returns what `JSON.stringify` returns, including `undefined` for a value it cannot represent.
 */
export function serializePayload(value: unknown): string {
  const json = JSON.stringify(value);

  if (json === undefined || !json.includes('\\ud')) {
    return json;
  }

  return JSON.stringify(value, (_key, v) => (typeof v === 'string' ? v.toWellFormed() : v));
}

import { type DescMessage, type MessageShape, fromBinary, toBinary } from '@bufbuild/protobuf';

type Builtin =
  Date | ((...args: never[]) => unknown) | Uint8Array | string | number | boolean | undefined;

/**
 * The partial message shape the SDK's clients accept, matching the `DeepPartial` type each
 * `src/protoc` module generates for its own messages.
 */
export type DeepPartial<T> = T extends Builtin
  ? T
  : T extends globalThis.Array<infer U>
    ? globalThis.Array<DeepPartial<U>>
    : T extends ReadonlyArray<infer U>
      ? ReadonlyArray<DeepPartial<U>>
      : T extends {}
        ? { [K in keyof T]?: DeepPartial<T[K]> }
        : Partial<T>;

/**
 * The part of a `ts-proto` message codec (what `src/protoc` exports under a message's name)
 * the bridge needs: the wire codec and the partial-to-full normalizer.
 */
export interface TsProtoCodec<T> {
  encode(message: T): { finish(): Uint8Array };
  decode(input: Uint8Array): T;
  fromPartial(object: DeepPartial<T>): T;
}

/**
 * Converts a `ts-proto` message to the protobuf-es message a Connect client sends. The two
 * bindings share the wire format, so an encode/decode round trip is an exact translation and
 * the SDK keeps its `ts-proto` types as the public request types.
 */
export function toProtobufEs<T, S extends DescMessage>(
  schema: S,
  codec: TsProtoCodec<T>,
  message: DeepPartial<T>
): MessageShape<S> {
  return fromBinary(schema, codec.encode(codec.fromPartial(message)).finish());
}

/**
 * Converts a protobuf-es message a Connect client received to the `ts-proto` message callers
 * expect, the inverse of `toProtobufEs`.
 */
export function fromProtobufEs<T, S extends DescMessage>(
  codec: TsProtoCodec<T>,
  schema: S,
  message: MessageShape<S>
): T {
  return codec.decode(toBinary(schema, message));
}

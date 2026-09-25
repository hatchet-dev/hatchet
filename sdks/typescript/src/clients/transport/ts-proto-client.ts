import type { DescMessage, DescService } from '@bufbuild/protobuf';
import {
  Code,
  ConnectError,
  createClient,
  type CallOptions as ConnectCallOptions,
  type Transport,
} from '@connectrpc/connect';
import { AbortError } from 'abort-controller-x';
import type { Client, TsProtoServiceDefinition } from 'nice-grpc';
import { Metadata, type CallOptions } from 'nice-grpc-common';
import { fromProtobufEs, toProtobufEs, type TsProtoCodec } from './message-bridge';

/**
 * The per-call options the Connect-backed clients accept: `nice-grpc`'s `CallOptions`
 * (`signal`, `metadata`, `onHeader`, `onTrailer`) plus the `deadline` its deadline middleware
 * adds, an absolute time as a `Date` or epoch milliseconds.
 */
export interface UnaryCallOptionsExt {
  deadline?: Date | number;
}

export type UnaryCallOptions = CallOptions & UnaryCallOptionsExt;

/**
 * The `ts-proto` message codec a service definition names for each method's messages: the
 * generated definitions carry `fromPartial`, which the bridge relies on to normalize a caller's
 * partial request.
 */
type MethodCodec = TsProtoCodec<unknown>;

/**
 * Builds the client a `ts-proto` service definition describes, with every method served by a
 * Connect client for the matching protobuf-es service on `transport`. The result has the exact
 * surface `nice-grpc`'s `ClientFactory.create(definition, channel)` produced: every RPC of the
 * service as `(request, options?) => Promise<response>` with the SDK's `ts-proto` message types,
 * so a caller holding one of the public client properties sees no difference.
 *
 * Both bindings are generated from the same proto, so a method's client name is the same in
 * each and the protobuf-es descriptor can be looked up by it.
 */
export function createTsProtoClient<Def extends TsProtoServiceDefinition>(
  definition: Def,
  service: DescService,
  transport: Transport
): Client<Def, UnaryCallOptionsExt> {
  const connectClient = createClient(service, transport) as Record<
    string,
    (input: unknown, options?: ConnectCallOptions) => Promise<unknown>
  >;

  const client: Record<string, unknown> = {};

  for (const [key, method] of Object.entries(definition.methods)) {
    const desc = service.method[key];
    if (!desc) {
      throw new Error(
        `${definition.fullName}.${method.name} has no protobuf-es descriptor; regenerate src/protoc-es`
      );
    }

    if (method.requestStream || method.responseStream) {
      client[key] = () => {
        throw new Error(
          `${definition.fullName}.${method.name} is a streaming RPC, which the unary transport does not serve`
        );
      };
      continue;
    }

    const requestCodec = method.requestType as MethodCodec;
    const responseCodec = method.responseType as MethodCodec;

    client[key] = (request: unknown, options?: UnaryCallOptions) =>
      unaryCall(
        () =>
          connectClient[key](
            toProtobufEs(desc.input, requestCodec, request as never),
            toConnectCallOptions(options)
          ),
        responseCodec,
        desc.output,
        options
      );
  }

  return client as Client<Def, UnaryCallOptionsExt>;
}

/**
 * Runs one unary call with `nice-grpc`'s cancellation and deadline semantics: a signal that
 * is already aborted rejects before anything is sent, an abort during the call rejects with the
 * signal's reason (an `AbortError`) rather than the transport's cancellation error, and a
 * deadline that has already passed rejects with `DEADLINE_EXCEEDED` without sending.
 */
async function unaryCall<Res, S extends DescMessage>(
  send: () => Promise<unknown>,
  responseCodec: TsProtoCodec<Res>,
  responseSchema: S,
  options?: UnaryCallOptions
): Promise<Res> {
  const signal = options?.signal;
  if (signal?.aborted) {
    throw abortReason(signal);
  }
  if (options?.deadline !== undefined && remainingMs(options.deadline) <= 0) {
    throw new ConnectError('the operation timed out', Code.DeadlineExceeded);
  }

  let response: unknown;
  try {
    response = await send();
  } catch (e) {
    if (signal?.aborted) {
      throw abortReason(signal);
    }
    throw e;
  }

  return fromProtobufEs(responseCodec, responseSchema, response as never);
}

function abortReason(signal: AbortSignal): unknown {
  return signal.reason ?? new AbortError();
}

function remainingMs(deadline: Date | number): number {
  const at = deadline instanceof Date ? deadline.getTime() : deadline;
  return at - Date.now();
}

/**
 * Translates `nice-grpc` call options to Connect's: the signal passes through, an absolute
 * deadline becomes the remaining time, request metadata becomes headers and the header and
 * trailer callbacks receive `Metadata` built from the response headers.
 */
export function toConnectCallOptions(options?: UnaryCallOptions): ConnectCallOptions | undefined {
  if (!options) {
    return undefined;
  }

  const { signal, deadline, metadata, onHeader, onTrailer } = options;
  const connect: ConnectCallOptions = {};

  if (signal) {
    connect.signal = signal;
  }
  if (deadline !== undefined) {
    connect.timeoutMs = Math.max(1, remainingMs(deadline));
  }
  if (metadata) {
    connect.headers = metadataToHeaders(metadata);
  }
  if (onHeader) {
    connect.onHeader = (headers) => onHeader(headersToMetadata(headers));
  }
  if (onTrailer) {
    connect.onTrailer = (headers) => onTrailer(headersToMetadata(headers));
  }

  return connect;
}

/**
 * gRPC metadata as HTTP headers: binary values (keys ending in `-bin`) are base64 encoded,
 * which is how they travel on the wire in every gRPC implementation.
 */
export function metadataToHeaders(metadata: Metadata): Headers {
  const headers = new Headers();
  for (const [key, values] of metadata) {
    for (const value of values) {
      headers.append(
        key,
        typeof value === 'string' ? value : Buffer.from(value).toString('base64')
      );
    }
  }
  return headers;
}

/**
 * Response headers as gRPC metadata, the inverse of `metadataToHeaders`. `Headers` joins
 * repeated values with a comma, so a binary header is split back into its values before each
 * is decoded.
 */
export function headersToMetadata(headers: Headers): Metadata {
  const metadata = Metadata();
  headers.forEach((value, key) => {
    if (key.endsWith('-bin')) {
      metadata.set(
        key,
        value.split(',').map((part) => new Uint8Array(Buffer.from(part.trim(), 'base64')))
      );
    } else {
      metadata.set(key, value);
    }
  });
  return metadata;
}

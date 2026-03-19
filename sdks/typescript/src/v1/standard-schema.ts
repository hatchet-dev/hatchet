/**
 * Standard Schema support for the Hatchet TypeScript SDK.
 *
 * Standard Schema (https://github.com/standard-schema/standard-schema) is a common
 * interface implemented by Zod v3.24+, Zod v4, Valibot, ArkType, and other validation
 * libraries. By accepting Standard Schema, the SDK is not locked to any specific
 * validation library.
 *
 * @module StandardSchema
 */

import { z } from 'zod';
import { zodToJsonSchema } from 'zod-to-json-schema';

// ---------------------------------------------------------------------------
// Standard Schema v1 types (inlined from @standard-schema/spec to avoid a
// runtime dependency – these are pure type-level declarations).
// See: https://github.com/standard-schema/standard-schema
// ---------------------------------------------------------------------------

/** The Standard Schema v1 interface. */
export interface StandardSchemaV1<Input = unknown, Output = Input> {
  readonly '~standard': StandardSchemaV1.Props<Input, Output>;
}

export namespace StandardSchemaV1 {
  /** The Standard Schema v1 properties interface. */
  export interface Props<Input = unknown, Output = Input> {
    readonly version: 1;
    readonly vendor: string;
    readonly validate: (
      value: unknown
    ) => Result<Output> | Promise<Result<Output>>;
    readonly types?: Types<Input, Output> | undefined;
  }

  /** The result type of the validate function. */
  export type Result<Output> = Output | FailureResult;

  /** The failure result type. */
  export interface FailureResult {
    readonly issues: ReadonlyArray<Issue>;
  }

  /** An issue from validation. */
  export interface Issue {
    readonly message: string;
    readonly path?: ReadonlyArray<
      PropertyKey | PathSegment
    >;
  }

  /** A path segment. */
  export interface PathSegment {
    readonly key: PropertyKey;
  }

  /** The Standard Schema v1 types interface. */
  export interface Types<Input = unknown, Output = Input> {
    readonly input?: Input;
    readonly output?: Output;
  }

  /** Infer the input type of a Standard Schema. */
  export type InferInput<Schema extends StandardSchemaV1> =
    NonNullable<Schema['~standard']['types']>['input'];

  /** Infer the output type of a Standard Schema. */
  export type InferOutput<Schema extends StandardSchemaV1> =
    NonNullable<Schema['~standard']['types']>['output'];
}

// ---------------------------------------------------------------------------
// Type guards
// ---------------------------------------------------------------------------

/** Returns true if the value implements the Standard Schema v1 interface. */
export function isStandardSchema(value: unknown): value is StandardSchemaV1 {
  return (
    typeof value === 'object' &&
    value !== null &&
    '~standard' in value &&
    typeof (value as any)['~standard'] === 'object'
  );
}

/** Returns true if the value is a Zod schema (has _def, used for zodToJsonSchema). */
export function isZodSchema(value: unknown): value is z.ZodType<any> {
  return (
    typeof value === 'object' &&
    value !== null &&
    '_def' in value &&
    typeof (value as any).parse === 'function'
  );
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

/**
 * A validation error thrown when a Standard Schema validation fails.
 */
export class StandardSchemaValidationError extends Error {
  public readonly issues: ReadonlyArray<StandardSchemaV1.Issue>;

  constructor(issues: ReadonlyArray<StandardSchemaV1.Issue>) {
    const message = issues.map((i) => i.message).join('; ');
    super(`Validation failed: ${message}`);
    this.name = 'StandardSchemaValidationError';
    this.issues = issues;
  }
}

function isFailureResult(
  result: StandardSchemaV1.Result<unknown>
): result is StandardSchemaV1.FailureResult {
  return (
    typeof result === 'object' &&
    result !== null &&
    'issues' in result &&
    Array.isArray((result as any).issues)
  );
}

/**
 * Validate data against either a Zod schema or a Standard Schema.
 *
 * - For Zod schemas (detected via `_def`), uses `.parse()` directly.
 * - For Standard Schema, uses `~standard.validate()`.
 */
export async function validateWithSchema<T = unknown>(
  schema: StandardSchemaV1<unknown, T> | z.ZodType<T>,
  data: unknown
): Promise<T> {
  // Prefer Zod's synchronous .parse() when available
  if (isZodSchema(schema)) {
    return schema.parse(data) as T;
  }

  if (isStandardSchema(schema)) {
    const result = await schema['~standard'].validate(data);
    if (isFailureResult(result)) {
      throw new StandardSchemaValidationError(result.issues);
    }
    return result as T;
  }

  throw new Error('inputValidator must be a Zod schema or Standard Schema v1 compliant object');
}

// ---------------------------------------------------------------------------
// JSON Schema conversion
// ---------------------------------------------------------------------------

/**
 * Convert a schema to JSON Schema if possible.
 *
 * - Zod schemas are converted using `zod-to-json-schema`.
 * - Non-Zod Standard Schema schemas return `undefined` since there is no
 *   universal Standard Schema → JSON Schema converter.
 */
export function schemaToJsonSchema(schema: unknown): object | undefined {
  if (isZodSchema(schema)) {
    return zodToJsonSchema(schema);
  }
  return undefined;
}

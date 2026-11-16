import { z } from 'zod';

const positiveInt = z.int().positive();

/** Tags on a sandbox. */
export const metadata = z.record(z.string(), z.string());

/** Machine size as the provider receives it. Sizes are MiB. */
export const resources = z.strictObject({
  vcpu: positiveInt.optional(),
  memoryMib: positiveInt.optional(),
  diskMib: positiveInt.optional(),
});

/** Input of one sandbox task run. `fn` names the operation, such as `$create`, `$command` or `$fs.read`. */
export const turnInput = z.strictObject({
  fn: z.string().min(1),
  input: z.record(z.string(), z.unknown()),
  sandboxId: z.string().min(1).optional(),
  image: z.string().min(1).optional(),
  snapshotId: z.string().min(1).optional(),
  metadata: metadata.optional(),
  idleTtlMs: positiveInt.optional(),
  inheritMetadata: z.boolean().optional(),
  resources: resources.optional(),
});
export type TurnInput = z.input<typeof turnInput>;

/** Output of one sandbox task run. `output` is the operation's own result. */
export const turnResult = z.looseObject({
  output: z.looseObject({}),
  sandboxId: z.string(),
  snapshotId: z.string().optional(),
  timings: z.record(z.string(), z.number()).optional(),
});
export type TurnResult = z.infer<typeof turnResult>;

/** `output` of a `$command` turn. */
export const commandOutput = z.looseObject({
  stdout: z.string(),
  stderr: z.string(),
  exitCode: z.int(),
});

/** One row of a `$list` turn's `output.sandboxes`. */
export const sandboxInfo = z.looseObject({
  sandboxId: z.string(),
  metadata: metadata.optional(),
  live: z.boolean(),
  latestSnapshot: z.string().optional(),
  createdAt: z.coerce.date(),
  lastUsed: z.coerce.date().optional(),
});

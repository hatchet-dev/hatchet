import { z } from 'zod';

/* ============================================================================
 * Validation Schemas
 * ========================================================================= */

// Base schemas for create (all fields required) and edit (most fields optional)
const createBaseSchema = z.object({
  clientId: z.string().min(1, 'Client ID is required'),
  clientSecret: z.string().min(1, 'Client Secret is required'),
  usesPkce: z.boolean(),
});

const editBaseSchema = z.object({
  clientId: z.string().min(1, 'Client ID is required'),
  clientSecret: z.string().optional(), // Optional in edit mode since API doesn't return it
  usesPkce: z.boolean(),
});

// Create schemas (all fields required)
const createOktaSchema = createBaseSchema.extend({
  provider: z.literal('Okta'),
  ssoDomain: z.string().min(1, 'SSO Domain is required'),
});

const createEntraSchema = createBaseSchema.extend({
  provider: z.literal('MicrosoftEntra'),
  tenantId: z.string().min(1, 'Tenant ID is required'),
});

const createGenericUrlFields = {
  authUrl: z.string().url('Must be a valid URL'),
  tokenUrl: z.string().url('Must be a valid URL'),
  userinfoUrl: z.string().url('Must be a valid URL'),
};

const createGenericSchema = createBaseSchema.extend({
  provider: z.literal('Generic'),
  ...createGenericUrlFields,
});

const createGoogleSchema = createBaseSchema.extend({
  provider: z.literal('Google'),
  ...createGenericUrlFields,
});

const createOneLoginSchema = createBaseSchema.extend({
  provider: z.literal('OneLogin'),
  ...createGenericUrlFields,
});

const createJumpCloudSchema = createBaseSchema.extend({
  provider: z.literal('JumpCloud'),
  ...createGenericUrlFields,
});

// Edit schemas (clientSecret optional)
const editOktaSchema = editBaseSchema.extend({
  provider: z.literal('Okta'),
  ssoDomain: z.string().min(1, 'SSO Domain is required'),
});

const editEntraSchema = editBaseSchema.extend({
  provider: z.literal('MicrosoftEntra'),
  tenantId: z.string().min(1, 'Tenant ID is required'),
});

const editGenericUrlFields = {
  authUrl: z.string().url('Must be a valid URL'),
  tokenUrl: z.string().url('Must be a valid URL'),
  userinfoUrl: z.string().url('Must be a valid URL'),
};

const editGenericSchema = editBaseSchema.extend({
  provider: z.literal('Generic'),
  ...editGenericUrlFields,
});

const editGoogleSchema = editBaseSchema.extend({
  provider: z.literal('Google'),
  ...editGenericUrlFields,
});

const editOneLoginSchema = editBaseSchema.extend({
  provider: z.literal('OneLogin'),
  ...editGenericUrlFields,
});

const editJumpCloudSchema = editBaseSchema.extend({
  provider: z.literal('JumpCloud'),
  ...editGenericUrlFields,
});

export const createFormSchema = z.discriminatedUnion('provider', [
  createOktaSchema,
  createEntraSchema,
  createGenericSchema,
  createGoogleSchema,
  createOneLoginSchema,
  createJumpCloudSchema,
]);
export const editFormSchema = z.discriminatedUnion('provider', [
  editOktaSchema,
  editEntraSchema,
  editGenericSchema,
  editGoogleSchema,
  editOneLoginSchema,
  editJumpCloudSchema,
]);
export type FormValues = z.infer<typeof createFormSchema>;
export type EditFormValues = z.infer<typeof editFormSchema>;

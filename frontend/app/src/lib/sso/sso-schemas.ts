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

const createGenericSchema = createBaseSchema.extend({
  provider: z.union([
    z.literal('Generic'),
    z.literal('Google'),
    z.literal('OneLogin'),
    z.literal('JumpCloud'),
  ]),
  authUrl: z.string().url('Must be a valid URL'),
  tokenUrl: z.string().url('Must be a valid URL'),
  userinfoUrl: z.string().url('Must be a valid URL'),
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

const editGenericSchema = editBaseSchema.extend({
  provider: z.union([
    z.literal('Generic'),
    z.literal('Google'),
    z.literal('OneLogin'),
    z.literal('JumpCloud'),
  ]),
  authUrl: z.string().url('Must be a valid URL'),
  tokenUrl: z.string().url('Must be a valid URL'),
  userinfoUrl: z.string().url('Must be a valid URL'),
});

export const createFormSchema = z.discriminatedUnion('provider', [
  createOktaSchema,
  createEntraSchema,
  createGenericSchema,
]);
export const editFormSchema = z.discriminatedUnion('provider', [
  editOktaSchema,
  editEntraSchema,
  editGenericSchema,
]);
export type FormValues = z.infer<typeof createFormSchema>;
export type EditFormValues = z.infer<typeof editFormSchema>;
